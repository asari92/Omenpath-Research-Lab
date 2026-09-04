package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func countOpenForTest(portals []domain.Portal) int {
	count := 0
	for _, portal := range portals {
		if portal.Status == domain.PortalStatusOpen {
			count++
		}
	}
	return count
}

func TestSimulationTick_FullOrderPortalObserverExtractionSpawnAttention(t *testing.T) {
	closing := tickNaturalClosePortal(1, 1, 4*time.Second)
	extraction := tickExtractionPortal(2, 1, 2, testutil.BaseTime)
	state := extractionTickState(
		[]domain.Portal{closing, extraction},
		tickOutboundObserver(1, 8*time.Second),
		tickExploringObserver(2, 5*time.Second),
	)
	state.NextPortalID = 3
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 5*time.Second)
	rnd := &recordingRandom{ints: []int{5, 0, 10, 0, 2}, floats: []float64{10, .1, .9}}

	result, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
	require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
	require.Equal(t, domain.ObserverReturning, state.Observers[1].Status)
	require.True(t, result.Spawned)
	require.Equal(t, int64(3), result.SpawnedPortalID)
	require.True(t, result.HasNeedsAttention)
	index, ok, err := domain.NeedsAttentionPortalIndex(state.Portals, testutil.BaseTime.Add(5*time.Second), config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, state.Portals[index].ID, result.NeedsAttentionPortalID)
}

func TestSimulationTick_NeedsAttentionUsesPostSpawnState(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.True(t, result.HasNeedsAttention)
	require.Equal(t, result.SpawnedPortalID, result.NeedsAttentionPortalID)
}

func TestSimulationTick_TerminalReleaseAllowsOnlyFreshSchedule(t *testing.T) {
	state := fullSimulationState()
	state.Portals[6].ScheduledCloseAt = testutil.BaseTime.Add(time.Second)
	state.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), testutil.NewFakeRandom().QueueInt(0), config.Default())
	require.NoError(t, err)
	require.False(t, result.Spawned)
	require.Equal(t, 6, countOpenForTest(state.Portals))
	require.False(t, state.NaturalSpawn.Paused)
	require.Equal(t, *state.NaturalSpawn.ScheduledAt, *state.NaturalSpawn.DueAt)
}

func TestSimulationTick_ZeroDelayProducesAtMostOneSpawn(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	result, err := state.ResolveTick(now, naturalSpawnRandom(0, 10, 0, 0, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	result, err = state.ResolveTick(now, nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Spawned)
	require.Len(t, state.Portals, 1)
}

func TestSimulationTick_SevenOpenPortalsNeverBecomeEight(t *testing.T) {
	state := fullSimulationState()
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 0)
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Spawned)
	require.Len(t, state.Portals, 7)
	require.Equal(t, 7, countOpenForTest(state.Portals))
	require.True(t, state.NaturalSpawn.Paused)
}

func TestSimulationTick_SameTimestampReplayIsNoOp(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	_, err := state.ResolveTick(now, nil, config.Default())
	require.NoError(t, err)
	before := state
	result, err := state.ResolveTick(now, nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, state)
}

func TestSimulationTick_SameTimestampReplayConsumesNoRandom(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	_, err := state.ResolveTick(now, naturalSpawnRandom(0, 10, 0, 0, 10, .1, .9), config.Default())
	require.NoError(t, err)
	before := state
	result, err := state.ResolveTick(now, nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, state)
}

func TestSimulationTick_BackwardTimeRejectsAtomically(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	last := testutil.BaseTime.Add(2 * time.Second)
	state.LastTickAt = &last
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_LargeJumpDoesNotBackfillNaturalSpawns(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	rnd := naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9)
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Minute), rnd, config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.Len(t, state.Portals, 1)
}

func TestSimulationTick_LargeJumpStillCatchesUpDomainLifecycles(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, 4*time.Second)
	state := observerTickState(p, tickOutboundObserver(1, 8*time.Second))
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 2*time.Minute)
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Minute), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
	require.Equal(t, domain.ObserverLost, state.Observers[0].Status)
}

func TestSimulationTick_RandomOrderExtractionBeforeNatural(t *testing.T) {
	state := extractionTickState([]domain.Portal{tickExtractionPortal(1, 1, 1, testutil.BaseTime)}, tickWaitingObserver(1, 0))
	state.NextPortalID = 2
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 5*time.Second)
	rnd := &recordingRandom{ints: []int{5, 0, 10, 0, 2}, floats: []float64{10, .1, .9}}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Equal(t, "int:5:15", rnd.calls[0])
	require.Equal(t, "int:0:84", rnd.calls[1])
}

func TestSimulationTick_InvalidStateConsumesNoRandom(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Planes = state.Planes[:84]
	rnd := &countingRandom{}
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestSimulationTick_InvalidStateDoesNotPartiallyResolvePortal(t *testing.T) {
	valid := tickNaturalClosePortal(1, 1, time.Second)
	invalid := tickNaturalClosePortal(2, 2, time.Minute)
	invalid.Status = domain.PortalStatus("INVALID")
	state := portalTickState(valid, invalid)
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_InvalidStateDoesNotPartiallyResolveObserver(t *testing.T) {
	state := observerTickState(observerPortal(), tickOutboundObserver(1, time.Second))
	unknown := int64(999)
	state.Observers[1] = tickExploringObserver(2, time.Second)
	state.Observers[1].CurrentPlaneID = &unknown
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_InvalidStateDoesNotPartiallySynchronizeExtraction(t *testing.T) {
	p := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
	wrong := testutil.BaseTime.Add(time.Second)
	p.ExtractionSynchronizedAt = &wrong
	state := extractionTickState([]domain.Portal{p}, tickExploringObserver(1, 5*time.Second))
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_InvalidStateDoesNotPartiallySpawn(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.NaturalSpawn.DueAt = nil
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_DoesNotPersistDerivedRealtimeValues(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, time.Minute)
	p.CreaturesInitial = 5
	state := portalTickState(p)
	state.Lab.EnergyBase = 20
	beforePortal := state.Portals[0]
	beforeLab := state.Lab
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, beforePortal, state.Portals[0])
	require.Equal(t, beforeLab, state.Lab)
}

func TestSimulationTick_DoesNotReorderAnyAggregateSlice(t *testing.T) {
	state := portalTickState(tickNaturalClosePortal(2, 2, time.Minute), tickNaturalClosePortal(1, 1, time.Minute))
	state.Planes[0], state.Planes[1] = state.Planes[1], state.Planes[0]
	state.Observers[0], state.Observers[1] = state.Observers[1], state.Observers[0]
	portalIDs := []int64{state.Portals[0].ID, state.Portals[1].ID}
	planeIDs := []int64{state.Planes[0].ID, state.Planes[1].ID}
	observerIDs := []int64{state.Observers[0].ID, state.Observers[1].ID}
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, portalIDs, []int64{state.Portals[0].ID, state.Portals[1].ID})
	require.Equal(t, planeIDs, []int64{state.Planes[0].ID, state.Planes[1].ID})
	require.Equal(t, observerIDs, []int64{state.Observers[0].ID, state.Observers[1].ID})
}

func TestSimulationTick_DoesNotCreateEvents(t *testing.T) {
	resultType := reflect.TypeOf(domain.SimulationTickResult{})
	_, hasEvents := resultType.FieldByName("Events")
	require.False(t, hasEvents)
}

func TestSimulationTick_NaturalPortalFactoryBehaviorUnchanged(t *testing.T) {
	cfg := config.Default()
	now := testutil.BaseTime.Add(time.Second)
	state := dueNaturalState(testutil.BaseTime)
	actualRandom := naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9)
	_, err := state.ResolveTick(now, actualRandom, cfg)
	require.NoError(t, err)
	expected := domain.NewNaturalPortal(1, 1, 1, now, cfg, &recordingRandom{
		ints: []int{10, 0}, floats: []float64{10, .1, .9},
	})
	require.Equal(t, expected, state.Portals[0])
}

func TestSimulationTick_ManualCommandsRemainOutsideTick(t *testing.T) {
	state := portalTickState(observerPortal())
	beforePortal := state.Portals[0]
	beforeObservers := append([]domain.Observer(nil), state.Observers...)
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, beforePortal, state.Portals[0])
	require.Equal(t, beforeObservers, state.Observers)
}

func TestSimulationTick_ResultIsDerivedFromCommittedState(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	result, err := state.ResolveTick(now, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.Equal(t, state.Lab.CurrentEnergy(now, config.Default()), result.LabEnergy)
	index, ok, err := domain.NeedsAttentionPortalIndex(state.Portals, now, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, state.Portals[index].ID, result.NeedsAttentionPortalID)
}

func TestSimulationTick_RejectsInvalidRequiredConfig(t *testing.T) {
	tests := map[string]func(*config.Config){
		"natural ttl":       func(cfg *config.Config) { cfg.NaturalTTLMin = 0 },
		"portal energy":     func(cfg *config.Config) { cfg.PortalEnergyMin = cfg.PortalEnergyMax + 1 },
		"portal decay":      func(cfg *config.Config) { cfg.PortalDecayMin = 0 },
		"stability":         func(cfg *config.Config) { cfg.UnstableProbability = 2 },
		"creature transit":  func(cfg *config.Config) { cfg.CreatureTransit = 0 },
		"observer transit":  func(cfg *config.Config) { cfg.ObserverTransitMin = 0 },
		"research":          func(cfg *config.Config) { cfg.ResearchDuration = 0 },
		"extraction":        func(cfg *config.Config) { cfg.ExtractionSync = 0 },
		"emergency":         func(cfg *config.Config) { cfg.EmergencyDuration = 0 },
		"attention horizon": func(cfg *config.Config) { cfg.RiskSafeHorizon = 0 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := config.Default()
			mutate(&cfg)
			state := simulationState(testutil.BaseTime)
			before := state
			_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, cfg)
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			require.Equal(t, before, state)
		})
	}
}

func TestSimulationTick_InvalidConfigConsumesNoRandom(t *testing.T) {
	cfg := config.Default()
	cfg.RiskSafeHorizon = 0
	state := dueNaturalState(testutil.BaseTime)
	before := state
	rnd := &countingRandom{}
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), rnd, cfg)
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}
