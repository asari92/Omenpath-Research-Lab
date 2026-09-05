package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func tickExtractionPortal(id int64, planeID int64, slot int, now time.Time) domain.Portal {
	p := domain.NewExtractionPortal(id, planeID, slot, now, config.Default(), extractionFactoryRandom(config.Default()))
	return p
}

func extractionTickState(portals []domain.Portal, observers ...domain.Observer) domain.SimulationState {
	state := simulationState(testutil.BaseTime)
	state.Portals = portals
	state.NextPortalID = int64(len(portals) + 1)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, time.Minute)
	state.Observers = domain.NewObserverRoster(config.Default().ObserverCount, testutil.BaseTime)
	for _, observer := range observers {
		state.Observers[observer.ID-1] = observer
	}
	return state
}

func TestSimulationTick_ExtractionBeforeSyncRemainsPending(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0))
	_, err := state.ResolveTick(testutil.BaseTime.Add(4*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Nil(t, state.Portals[0].ExtractionSynchronizedAt)
	require.Equal(t, domain.ObserverWaitingReturn, state.Observers[0].Status)
}

func TestSimulationTick_ExtractionAtSyncCompletes(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Portals[0].ExtractionSynchronizedAt)
}

func TestSimulationTick_ResearchCompletionFeedsSameTickExtraction(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickExploringObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Observers[0].PhaseStartedAt)
}

func TestSimulationTick_ExtractionSelectsCurrentLongestWaiting(t *testing.T) {
	older := tickWaitingObserver(2, -time.Minute)
	newer := tickWaitingObserver(1, -30*time.Second)
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, newer, older)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
}

func TestSimulationTick_ExtractionOriginalCandidateGoneSelectsNext(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(2, -time.Minute))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverAvailable, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
}

func TestSimulationTick_ExtractionWithoutWaitingCompletesWithoutReturn(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)})
	rnd := &countingRandom{}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.NotNil(t, state.Portals[0].ExtractionSynchronizedAt)
	require.Zero(t, rnd.intCalls)
}

func TestSimulationTick_CompletedExtractionDoesNotReturnSecondObserver(t *testing.T) {
	state := extractionTickState(
		[]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)},
		tickWaitingObserver(1, -time.Minute), tickWaitingObserver(2, -30*time.Second),
	)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	_, err = state.ResolveTick(testutil.BaseTime.Add(6*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverWaitingReturn, state.Observers[1].Status)
}

func TestSimulationTick_TerminalExtractionDoesNotSynchronize(t *testing.T) {
	p := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	closedAt := testutil.BaseTime.Add(time.Second)
	makeTerminalPortal(&p, domain.PortalStatusClosed, domain.TerminationManualClose, closedAt)
	state := extractionTickState([]domain.Portal{p}, tickWaitingObserver(1, 0))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Nil(t, state.Portals[0].ExtractionSynchronizedAt)
}

func TestSimulationTick_PortalCloseAtSyncWinsBeforeExtraction(t *testing.T) {
	p := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	p.ScheduledCloseAt = testutil.BaseTime.Add(5 * time.Second)
	state := extractionTickState([]domain.Portal{p}, tickWaitingObserver(1, 0))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
	require.Nil(t, state.Portals[0].ExtractionSynchronizedAt)
	require.Equal(t, domain.ObserverWaitingReturn, state.Observers[0].Status)
}

func TestSimulationTick_MultipleExtractionsVisitPortalIDOrder(t *testing.T) {
	higher := tickExtractionPortal(2, 1, 2, testutil.BaseTime)
	lower := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	state := extractionTickState([]domain.Portal{higher, lower}, tickWaitingObserver(1, -time.Minute), tickWaitingObserver(2, -30*time.Second))
	rnd := testutil.NewFakeRandom().QueueInt(5, 6)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
	require.Equal(t, int64(1), *state.Observers[0].ActivePortalID)
	require.Equal(t, int64(2), *state.Observers[1].ActivePortalID)
}

func TestSimulationTick_MultipleExtractionsReturnDifferentObservers(t *testing.T) {
	state := extractionTickState(
		[]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime), tickExtractionPortal(2, 1, 2, testutil.BaseTime)},
		tickWaitingObserver(1, -time.Minute), tickWaitingObserver(2, -30*time.Second),
	)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5, 6), config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
}

func TestSimulationTick_MultipleLateExtractionsDoNotRejectDueTransitCreatedInSameStage(t *testing.T) {
	state := extractionTickState(
		[]domain.Portal{
			tickExtractionPortal(1, 1, 1, testutil.BaseTime),
			tickExtractionPortal(2, 1, 2, testutil.BaseTime),
		},
		tickWaitingObserver(1, -time.Minute),
		tickWaitingObserver(2, -30*time.Second),
	)
	rnd := testutil.NewFakeRandom().QueueInt(5, 6)

	_, err := state.ResolveTick(testutil.BaseTime.Add(20*time.Second), rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
	require.Equal(t, int64(1), *state.Observers[0].ActivePortalID)
	require.Equal(t, int64(2), *state.Observers[1].ActivePortalID)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *state.Observers[0].PhaseEndsAt)
	require.Equal(t, testutil.BaseTime.Add(11*time.Second), *state.Observers[1].PhaseEndsAt)
}

func TestSimulationTick_ExtractionDrawsTransitDurationBeforeNaturalSpawn(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0))
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 5*time.Second)
	rnd := &recordingRandom{ints: []int{5, 0, 10, 0, 2}, floats: []float64{10, .1, .9}}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Equal(t, "int:5:15", rnd.calls[0])
	require.Equal(t, "int:0:84", rnd.calls[1])
}

func TestSimulationTick_ExtractionNoWaitConsumesNoTransitRandom(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)})
	rnd := &countingRandom{}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Zero(t, rnd.intCalls)
}

func TestSimulationTick_ExtractionMarkerUsesSemanticDeadline(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)})
	_, err := state.ResolveTick(testutil.BaseTime.Add(7*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.NotNil(t, state.Portals[0].ExtractionSynchronizedAt)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Portals[0].ExtractionSynchronizedAt)
}

func TestSimulationTick_ExtractionTransitUsesSemanticDeadline(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0))
	_, err := state.ResolveTick(testutil.BaseTime.Add(7*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.NotNil(t, state.Observers[0].PhaseStartedAt)
	require.NotNil(t, state.Observers[0].PhaseEndsAt)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Observers[0].PhaseStartedAt)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *state.Observers[0].PhaseEndsAt)
}

func TestSimulationTick_ExtractionDoesNotReorderPortals(t *testing.T) {
	higher := tickExtractionPortal(2, 1, 2, testutil.BaseTime)
	lower := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	state := extractionTickState([]domain.Portal{higher, lower})
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, []int64{2, 1}, []int64{state.Portals[0].ID, state.Portals[1].ID})
}

func TestSimulationTick_ExtractionDoesNotReorderObservers(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0), tickWaitingObserver(2, time.Second))
	state.Observers[0], state.Observers[1] = state.Observers[1], state.Observers[0]
	before := []int64{state.Observers[0].ID, state.Observers[1].ID}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, before, []int64{state.Observers[0].ID, state.Observers[1].ID})
}

func TestSimulationTick_InvalidExtractionAggregateIsAtomic(t *testing.T) {
	p := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	wrong := testutil.BaseTime.Add(time.Second)
	p.ExtractionSynchronizedAt = &wrong
	state := extractionTickState([]domain.Portal{p}, tickExploringObserver(1, 5*time.Second))
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}
