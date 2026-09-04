package domain_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

type countingRandom struct {
	intValue   int
	intCalls   int
	floatCalls int
}

func (r *countingRandom) IntInclusive(_, _ int) int {
	r.intCalls++
	return r.intValue
}

func (r *countingRandom) FloatRange(_, _ float64) float64 {
	r.floatCalls++
	return 0
}

func simulationPlanes() []domain.Plane {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: fmt.Sprintf("Plane %02d", i+1)}
	}
	return planes
}

func scheduledSpawn(scheduledAt time.Time, delay time.Duration) domain.NaturalSpawnState {
	dueAt := scheduledAt.Add(delay)
	return domain.NaturalSpawnState{ScheduledAt: &scheduledAt, DueAt: &dueAt}
}

func simulationState(now time.Time) domain.SimulationState {
	return domain.SimulationState{
		Lab:          domain.LabState{EnergyBase: 100, EnergyBaseAt: now},
		Planes:       simulationPlanes(),
		Observers:    domain.NewObserverRoster(10, now),
		NextPortalID: 1,
		NaturalSpawn: scheduledSpawn(now, 10*time.Second),
	}
}

func resolveStateForValidation(state *domain.SimulationState, now time.Time, rnd *countingRandom, cfg config.Config) error {
	_, err := state.ResolveTick(now, rnd, cfg)
	return err
}

func TestNewNaturalSpawnState_AcceptsZeroDelay(t *testing.T) {
	cfg := config.Default()
	spawn, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, testutil.NewFakeRandom().QueueInt(0))
	require.NoError(t, err)
	require.False(t, spawn.Paused)
	require.Equal(t, testutil.BaseTime, *spawn.ScheduledAt)
	require.Equal(t, testutil.BaseTime, *spawn.DueAt)
}

func TestNewNaturalSpawnState_AcceptsMaximumDelay(t *testing.T) {
	cfg := config.Default()
	spawn, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, testutil.NewFakeRandom().QueueInt(20))
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(20*time.Second), *spawn.DueAt)
}

func TestNewNaturalSpawnState_DrawsExactlyOnce(t *testing.T) {
	rnd := &countingRandom{intValue: 7}
	_, err := domain.NewNaturalSpawnState(testutil.BaseTime, config.Default(), rnd)
	require.NoError(t, err)
	require.Equal(t, 1, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestNewNaturalSpawnState_StoresSchedulingOrigin(t *testing.T) {
	spawn, err := domain.NewNaturalSpawnState(testutil.BaseTime, config.Default(), testutil.NewFakeRandom().QueueInt(3))
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime, *spawn.ScheduledAt)
}

func TestNewNaturalSpawnState_RejectsNegativeMinimum(t *testing.T) {
	cfg := config.Default()
	cfg.SpawnDelayMin = -time.Second
	_, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, &countingRandom{})
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestNewNaturalSpawnState_RejectsReversedBounds(t *testing.T) {
	cfg := config.Default()
	cfg.SpawnDelayMin = 21 * time.Second
	_, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, &countingRandom{})
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestNewNaturalSpawnState_RejectsSubsecondBounds(t *testing.T) {
	cfg := config.Default()
	cfg.SpawnDelayMin = 500 * time.Millisecond
	_, err := domain.NewNaturalSpawnState(testutil.BaseTime, cfg, &countingRandom{})
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestNewNaturalSpawnState_RejectsNilRandom(t *testing.T) {
	_, err := domain.NewNaturalSpawnState(testutil.BaseTime, config.Default(), nil)
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsNilState(t *testing.T) {
	var state *domain.SimulationState
	_, err := state.ResolveTick(testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RequiresExactlyEightyFivePlanes(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.Planes = state.Planes[:84]
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsDuplicatePlaneIDs(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.Planes[84].ID = state.Planes[0].ID
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsNonCanonicalPlaneExploration(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*domain.Plane)
	}{
		{"explored without timestamp", func(p *domain.Plane) { p.Explored = true }},
		{"unexplored with timestamp", func(p *domain.Plane) {
			at := testutil.BaseTime
			p.ExploredAt = &at
		}},
		{"future exploration", func(p *domain.Plane) {
			at := testutil.BaseTime.Add(time.Second)
			p.Explored = true
			p.ExploredAt = &at
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := simulationState(testutil.BaseTime)
			tt.mutate(&state.Planes[0])
			before := state
			rnd := &countingRandom{}

			err := resolveStateForValidation(&state, testutil.BaseTime, rnd, config.Default())

			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			require.Equal(t, before, state)
			require.Zero(t, rnd.intCalls)
			require.Zero(t, rnd.floatCalls)
		})
	}
}

func TestSimulationState_RejectsDuplicatePortalIDs(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	p1 := testutil.NewPortalBuilder().Build()
	p2 := testutil.NewPortalBuilder().Slot(2).Build()
	state.Portals = []domain.Portal{p1, p2}
	state.NextPortalID = 2
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsDuplicateOpenSlots(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	p1 := testutil.NewPortalBuilder().Build()
	p2 := testutil.NewPortalBuilder().Build()
	p2.ID = 2
	p2.Name = "Omenpath #0002"
	p2.DestinationPlaneID = 2
	state.Portals = []domain.Portal{p1, p2}
	state.NextPortalID = 3
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_AllowsTerminalSlotReuse(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	closedAt := testutil.BaseTime
	closed := testutil.NewPortalBuilder().Build()
	closed.Status = domain.PortalStatusClosed
	closed.TerminationReason = domain.TerminationManualClose
	closed.ClosedAt = &closedAt
	open := testutil.NewPortalBuilder().Build()
	open.ID = 2
	open.Name = "Omenpath #0002"
	open.DestinationPlaneID = 2
	state.Portals = []domain.Portal{closed, open}
	state.NextPortalID = 3
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.NoError(t, err)
}

func TestSimulationState_RejectsUnknownPortalStatus(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	p := testutil.NewPortalBuilder().Build()
	p.Status = domain.PortalStatus("BROKEN")
	state.Portals = []domain.Portal{p}
	state.NextPortalID = 2
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsOpenPortalWithClosedAt(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	p := testutil.NewPortalBuilder().Build()
	closedAt := testutil.BaseTime
	p.ClosedAt = &closedAt
	state.Portals = []domain.Portal{p}
	state.NextPortalID = 2
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsTerminalPortalWithoutClosedAt(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	p := testutil.NewPortalBuilder().Build()
	p.Status = domain.PortalStatusClosed
	p.TerminationReason = domain.TerminationManualClose
	state.Portals = []domain.Portal{p}
	state.NextPortalID = 2
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsInvalidNextPortalID(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.Portals = []domain.Portal{testutil.NewPortalBuilder().Build()}
	state.NextPortalID = 1
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RequiresConfiguredObserverCount(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.Observers = state.Observers[:9]
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsMalformedSpawnSchedule(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn.DueAt = nil
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_RejectsFutureSpawnScheduleOrigin(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime.Add(time.Second), time.Second)

	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())

	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestSimulationState_InvalidAggregateDoesNotConsumeRandom(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.Planes = nil
	rnd := &countingRandom{}
	err := resolveStateForValidation(&state, testutil.BaseTime, rnd, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestSimulationState_InvalidAggregateIsAtomic(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NextPortalID = 0
	before := state
	err := resolveStateForValidation(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}
