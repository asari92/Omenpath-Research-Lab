package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func observerPhase(id int64, status domain.ObserverStatus, planeID, portalID *int64, started, ends *time.Time) domain.Observer {
	return domain.Observer{
		ID: id, Status: status, CurrentPlaneID: planeID, ActivePortalID: portalID,
		PhaseStartedAt: started, PhaseEndsAt: ends,
		CreatedAt: testutil.BaseTime, UpdatedAt: testutil.BaseTime,
	}
}

func observerTime(after time.Duration) *time.Time {
	at := testutil.BaseTime.Add(after)
	return &at
}

func observerID(id int64) *int64 {
	value := id
	return &value
}

func observerTickState(portal domain.Portal, observers ...domain.Observer) domain.SimulationState {
	state := portalTickState(portal)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, time.Minute)
	state.Observers = domain.NewObserverRoster(10, testutil.BaseTime)
	for _, observer := range observers {
		state.Observers[observer.ID-1] = observer
	}
	return state
}

func tickOutboundObserver(id int64, endsAfter time.Duration) domain.Observer {
	return observerPhase(id, domain.ObserverOutbound, nil, observerID(1), observerTime(0), observerTime(endsAfter))
}

func tickExploringObserver(id int64, endsAfter time.Duration) domain.Observer {
	return observerPhase(id, domain.ObserverExploring, observerID(1), nil, observerTime(0), observerTime(endsAfter))
}

func tickWaitingObserver(id int64, startedAfter time.Duration) domain.Observer {
	return observerPhase(id, domain.ObserverWaitingReturn, observerID(1), nil, observerTime(startedAfter), nil)
}

func tickReturningObserver(id int64, endsAfter time.Duration) domain.Observer {
	return observerPhase(id, domain.ObserverReturning, observerID(1), observerID(1), observerTime(0), observerTime(endsAfter))
}

func observerPortal() domain.Portal {
	p := tickNaturalClosePortal(1, 1, time.Minute)
	p.ObserverFlow = domain.PortalFlowOutbound
	return p
}

func TestSimulationTick_OutboundBeforeDeadlineRemainsOutbound(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(4*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverOutbound, state.Observers[0].Status)
}

func TestSimulationTick_OutboundAtDeadlineStartsResearch(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverExploring, state.Observers[0].Status)
	require.Equal(t, testutil.BaseTime.Add(25*time.Second), *state.Observers[0].PhaseEndsAt)
}

func TestSimulationTick_LateOutboundCatchesUpToWaitingReturn(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(30*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverWaitingReturn, state.Observers[0].Status)
	require.Equal(t, testutil.BaseTime.Add(25*time.Second), *state.Observers[0].PhaseStartedAt)
}

func TestSimulationTick_ResearchAtDeadlineBecomesWaitingReturn(t *testing.T) {
	state := observerTickState(observerPortal(), tickExploringObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverWaitingReturn, state.Observers[0].Status)
}

func TestSimulationTick_ReturningAtDeadlineBecomesAvailable(t *testing.T) {
	p := observerPortal()
	p.ObserverFlow = domain.PortalFlowInbound
	state := observerTickState(p, tickReturningObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverAvailable, state.Observers[0].Status)
}

func TestSimulationTick_SuccessfulReturnExploresPlane(t *testing.T) {
	p := observerPortal()
	p.ObserverFlow = domain.PortalFlowInbound
	state := observerTickState(p, tickReturningObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.True(t, state.Planes[0].Explored)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Planes[0].ExploredAt)
}

func TestSimulationTick_RepeatReturnPreservesExploredAt(t *testing.T) {
	p := observerPortal()
	p.ObserverFlow = domain.PortalFlowInbound
	state := observerTickState(p, tickReturningObserver(1, 5*time.Second))
	original := testutil.BaseTime.Add(-time.Minute)
	state.Planes[0].Explored = true
	state.Planes[0].ExploredAt = &original
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, original, *state.Planes[0].ExploredAt)
}

func TestSimulationTick_PortalClosedBeforeTransitMakesObserverLost(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, 4*time.Second)
	state := observerTickState(p, tickOutboundObserver(1, 8*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
}

func TestSimulationTick_PortalCollapsedBeforeTransitMakesObserverLost(t *testing.T) {
	p := tickEnergyCollapsePortal(1, 1, 4*time.Second)
	state := observerTickState(p, tickOutboundObserver(1, 8*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
}

func TestSimulationTick_CloseAtTransitDeadlineAllowsArrival(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, 5*time.Second)
	state := observerTickState(p, tickOutboundObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverExploring, state.Observers[0].Status)
}

func TestSimulationTick_PortalLifecycleRunsBeforeObserverLifecycle(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, 4*time.Second)
	state := observerTickState(p, tickOutboundObserver(1, 8*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
	require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
}

func TestSimulationTick_EarliestSuccessfulReturnSetsExploredAtRegardlessOfObserverID(t *testing.T) {
	p := observerPortal()
	p.ObserverFlow = domain.PortalFlowInbound
	lowerIDLater := tickReturningObserver(1, 7*time.Second)
	higherIDEarlier := tickReturningObserver(2, 5*time.Second)
	state := observerTickState(p, lowerIDLater, higherIDEarlier)
	state.Observers[0], state.Observers[1] = state.Observers[1], state.Observers[0]

	_, err := state.ResolveTick(testutil.BaseTime.Add(7*time.Second), nil, config.Default())

	require.NoError(t, err)
	require.True(t, state.Planes[0].Explored)
	require.NotNil(t, state.Planes[0].ExploredAt)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *state.Planes[0].ExploredAt)
	require.Equal(t, domain.ObserverAvailable, state.Observers[0].Status)
	require.Equal(t, domain.ObserverAvailable, state.Observers[1].Status)
}

func TestSimulationTick_ObserverTraversalDoesNotReorderSlice(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second), tickOutboundObserver(2, 5*time.Second))
	state.Observers[0], state.Observers[1] = state.Observers[1], state.Observers[0]
	before := []int64{state.Observers[0].ID, state.Observers[1].ID}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, []int64{state.Observers[0].ID, state.Observers[1].ID})
}

func TestSimulationTick_MultipleObserversResolveIndependently(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second), tickOutboundObserver(2, 8*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverExploring, state.Observers[0].Status)
	require.Equal(t, domain.ObserverOutbound, state.Observers[1].Status)
}

func TestSimulationTick_WaitingObserverRemainsWaiting(t *testing.T) {
	state := observerTickState(observerPortal(), tickWaitingObserver(1, 5*time.Second))
	before := state.Observers[0]
	_, err := state.ResolveTick(testutil.BaseTime.Add(6*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Observers[0])
}

func TestSimulationTick_AvailableAndLostObserversRemainUnchanged(t *testing.T) {
	state := observerTickState(observerPortal())
	state.Observers[1].Status = domain.ObserverLost
	beforeAvailable, beforeLost := state.Observers[0], state.Observers[1]
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, beforeAvailable, state.Observers[0])
	require.Equal(t, beforeLost, state.Observers[1])
}

func TestSimulationTick_RejectsUnknownObserverPlaneReference(t *testing.T) {
	state := observerTickState(observerPortal(), tickExploringObserver(1, 5*time.Second))
	unknown := int64(999)
	state.Observers[0].CurrentPlaneID = &unknown
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_RejectsUnknownActivePortalReference(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	unknown := int64(999)
	state.Observers[0].ActivePortalID = &unknown
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_DueObserverStateIsValidPreflight(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, 5*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
}

func TestSimulationTick_ObserverStageFailureIsAtomic(t *testing.T) {
	state := observerTickState(observerPortal(), tickExploringObserver(1, 5*time.Second))
	unknown := int64(999)
	state.Observers[0].CurrentPlaneID = &unknown
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}
