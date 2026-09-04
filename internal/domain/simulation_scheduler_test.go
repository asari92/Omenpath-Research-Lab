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

func openPortals(n int) []domain.Portal {
	portals := make([]domain.Portal, n)
	for i := range portals {
		p := testutil.NewPortalBuilder().Slot(i + 1).Build()
		p.ID = int64(i + 1)
		p.Name = fmt.Sprintf("Omenpath #%04d", i+1)
		p.DestinationPlaneID = int64(i + 1)
		portals[i] = p
	}
	return portals
}

func sevenOpenPortals() []domain.Portal { return openPortals(7) }

func sixOpenAndOneTerminalPortals() []domain.Portal {
	portals := openPortals(6)
	closedAt := testutil.BaseTime
	terminal := testutil.NewPortalBuilder().Slot(7).Build()
	terminal.ID = 7
	terminal.Name = "Omenpath #0007"
	terminal.DestinationPlaneID = 7
	terminal.Status = domain.PortalStatusClosed
	terminal.TerminationReason = domain.TerminationManualClose
	terminal.ClosedAt = &closedAt
	return append(portals, terminal)
}

func fullSimulationState() domain.SimulationState {
	state := simulationState(testutil.BaseTime)
	state.Portals = sevenOpenPortals()
	state.NextPortalID = 8
	return state
}

func pausedSimulationWithFreeSlot() domain.SimulationState {
	state := simulationState(testutil.BaseTime)
	state.Portals = sixOpenAndOneTerminalPortals()
	state.NextPortalID = 8
	state.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	return state
}

func TestResolveNaturalSpawn_BeforeDeadlineIsNoOp(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	before := state
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.False(t, result.Spawned)
	require.Equal(t, before, state)
}

func TestResolveNaturalSpawn_AtSchedulingOriginIsNoOpForZeroDelay(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 0)
	before := state
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime, nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, state)
}

func TestResolveNaturalSpawn_FullCapacityPausesImmediately(t *testing.T) {
	state := fullSimulationState()
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.True(t, state.NaturalSpawn.Paused)
}

func TestResolveNaturalSpawn_FullCapacityClearsExistingSchedule(t *testing.T) {
	state := fullSimulationState()
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Nil(t, state.NaturalSpawn.ScheduledAt)
	require.Nil(t, state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_FullCapacityConsumesNoRandom(t *testing.T) {
	state := fullSimulationState()
	rnd := &countingRandom{}
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestResolveNaturalSpawn_AlreadyPausedAndFullIsIdempotent(t *testing.T) {
	state := fullSimulationState()
	state.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	before := state
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Changed)
	require.Equal(t, before, state)
}

func TestResolveNaturalSpawn_CountsOnlyOpenPortals(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	rnd := testutil.NewFakeRandom().QueueInt(4)
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.False(t, state.NaturalSpawn.Paused)
}

func TestResolveNaturalSpawn_TerminalSlotAllowsFreshSchedule(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(6*time.Second), *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_FirstTickAfterFreeOnlySchedules(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	beforePortals := append([]domain.Portal(nil), state.Portals...)
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), testutil.NewFakeRandom().QueueInt(3), config.Default())
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.False(t, result.Spawned)
	require.Equal(t, beforePortals, state.Portals)
}

func TestResolveNaturalSpawn_FirstTickAfterFreeDoesNotSpawnAtZeroDelay(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), testutil.NewFakeRandom().QueueInt(0), config.Default())
	require.NoError(t, err)
	require.False(t, result.Spawned)
	require.Len(t, state.Portals, 7)
	require.Equal(t, *state.NaturalSpawn.ScheduledAt, *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_FirstTickAfterFreeDrawsExactlyOnce(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	rnd := &countingRandom{intValue: 2}
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestResolveNaturalSpawn_FreshScheduleUsesCurrentTickAsOrigin(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	now := testutil.BaseTime.Add(7 * time.Second)
	_, err := domain.ResolveNaturalSpawn(&state, now, testutil.NewFakeRandom().QueueInt(2), config.Default())
	require.NoError(t, err)
	require.Equal(t, now, *state.NaturalSpawn.ScheduledAt)
	require.Equal(t, now.Add(2*time.Second), *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_FreshScheduleUsesInclusiveMaximum(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	now := testutil.BaseTime.Add(time.Second)
	_, err := domain.ResolveNaturalSpawn(&state, now, testutil.NewFakeRandom().QueueInt(20), config.Default())
	require.NoError(t, err)
	require.Equal(t, now.Add(20*time.Second), *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_PauseDoesNotChangePortalRecords(t *testing.T) {
	state := fullSimulationState()
	before := append([]domain.Portal(nil), state.Portals...)
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Portals)
}

func TestResolveNaturalSpawn_ResumeDoesNotChangePortalRecords(t *testing.T) {
	state := pausedSimulationWithFreeSlot()
	before := append([]domain.Portal(nil), state.Portals...)
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), testutil.NewFakeRandom().QueueInt(1), config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Portals)
}

func TestResolveNaturalSpawn_InvalidSchedulerIsAtomic(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn.DueAt = nil
	before := state
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, state)
}

func TestResolveNaturalSpawn_InvalidSchedulerConsumesNoRandom(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	state.NaturalSpawn.ScheduledAt = nil
	rnd := &countingRandom{}
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime, rnd, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestResolveNaturalSpawn_RejectsNilState(t *testing.T) {
	_, err := domain.ResolveNaturalSpawn(nil, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}
