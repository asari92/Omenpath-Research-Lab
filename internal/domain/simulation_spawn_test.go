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

type recordingRandom struct {
	ints   []int
	floats []float64
	calls  []string
}

func (r *recordingRandom) IntInclusive(min, max int) int {
	r.calls = append(r.calls, fmt.Sprintf("int:%d:%d", min, max))
	if len(r.ints) == 0 {
		panic("recordingRandom: no int")
	}
	v := r.ints[0]
	r.ints = r.ints[1:]
	return v
}

func (r *recordingRandom) FloatRange(min, max float64) float64 {
	r.calls = append(r.calls, fmt.Sprintf("float:%.1f:%.1f", min, max))
	if len(r.floats) == 0 {
		panic("recordingRandom: no float")
	}
	v := r.floats[0]
	r.floats = r.floats[1:]
	return v
}

func dueNaturalState(now time.Time) domain.SimulationState {
	state := simulationState(now)
	state.NaturalSpawn = scheduledSpawn(now, time.Second)
	return state
}

func naturalSpawnRandom(destination, ttl, creatures, nextDelay int, energy, decay, stability float64) *recordingRandom {
	return &recordingRandom{
		ints:   []int{destination, ttl, creatures, nextDelay},
		floats: []float64{energy, decay, stability},
	}
}

func resolveDueNatural(t *testing.T, state *domain.SimulationState, rnd *recordingRandom) domain.NaturalSpawnResult {
	t.Helper()
	result, err := domain.ResolveNaturalSpawn(state, testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.NotEmpty(t, state.Portals)
	return result
}

func TestResolveNaturalSpawn_AtDeadlineSpawnsOnePortal(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	result := resolveDueNatural(t, &state, naturalSpawnRandom(6, 10, 0, 12, 10, .1, .9))
	require.True(t, result.Changed)
	require.True(t, result.Spawned)
	require.Equal(t, int64(1), result.PortalID)
	require.Len(t, state.Portals, 1)
}

func TestResolveNaturalSpawn_LateTickSpawnsOnlyOnePortal(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	rnd := naturalSpawnRandom(0, 10, 0, 0, 10, .1, .9)
	result, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Minute), rnd, config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.Len(t, state.Portals, 1)
}

func TestResolveNaturalSpawn_UsesTickTimeAsOpenedAt(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(7 * time.Second)
	result, err := domain.ResolveNaturalSpawn(&state, now, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.Equal(t, now, state.Portals[0].OpenedAt)
}

func TestResolveNaturalSpawn_UsesFirstFreeSlot(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = openPortals(3)
	state.Portals[1].Status = domain.PortalStatusClosed
	closedAt := testutil.BaseTime
	state.Portals[1].ClosedAt = &closedAt
	state.Portals[1].TerminationReason = domain.TerminationManualClose
	state.NextPortalID = 4
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, 2, state.Portals[3].SlotIndex)
}

func TestResolveNaturalSpawn_ReusesTerminalPortalSlot(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = sixOpenAndOneTerminalPortals()
	state.NextPortalID = 8
	resolveDueNatural(t, &state, &recordingRandom{ints: []int{0, 10, 0}, floats: []float64{10, .1, .9}})
	require.Equal(t, 7, state.Portals[7].SlotIndex)
}

func TestResolveNaturalSpawn_SelectsFirstPlaneIndex(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, int64(1), state.Portals[0].DestinationPlaneID)
}

func TestResolveNaturalSpawn_SelectsLastPlaneIndex(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	resolveDueNatural(t, &state, naturalSpawnRandom(84, 10, 0, 2, 10, .1, .9))
	require.Equal(t, int64(85), state.Portals[0].DestinationPlaneID)
}

func TestResolveNaturalSpawn_AllowsRepeatedDestination(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = openPortals(1)
	state.Portals[0].DestinationPlaneID = 7
	state.NextPortalID = 2
	resolveDueNatural(t, &state, naturalSpawnRandom(6, 10, 0, 2, 10, .1, .9))
	require.Equal(t, int64(7), state.Portals[0].DestinationPlaneID)
	require.Equal(t, int64(7), state.Portals[1].DestinationPlaneID)
}

func TestResolveNaturalSpawn_UsesNextPortalID(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.NextPortalID = 42
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, int64(42), state.Portals[0].ID)
}

func TestResolveNaturalSpawn_FormatsSequentialPortalName(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.NextPortalID = 42
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, "Omenpath #0042", state.Portals[0].Name)
}

func TestResolveNaturalSpawn_IncrementsSequenceExactlyOnce(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.NextPortalID = 42
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, int64(43), state.NextPortalID)
}

func TestResolveNaturalSpawn_PreservesExistingPortalOrder(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = openPortals(2)
	state.Portals[0], state.Portals[1] = state.Portals[1], state.Portals[0]
	state.NextPortalID = 3
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, []int64{2, 1, 3}, []int64{state.Portals[0].ID, state.Portals[1].ID, state.Portals[2].ID})
}

func TestResolveNaturalSpawn_AppendsWithoutRemovingTerminalHistory(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = sixOpenAndOneTerminalPortals()
	state.NextPortalID = 8
	resolveDueNatural(t, &state, &recordingRandom{ints: []int{0, 10, 0}, floats: []float64{10, .1, .9}})
	require.Len(t, state.Portals, 8)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[6].Status)
}

func TestResolveNaturalSpawn_ReusesNaturalFactoryRanges(t *testing.T) {
	cfg := config.Default()
	state := dueNaturalState(testutil.BaseTime)
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 300, 0, 2, 100, 1, .9))
	p := state.Portals[0]
	require.Equal(t, cfg.PortalEnergyMax, p.EnergyBase)
	require.Equal(t, cfg.PortalDecayMax, p.EnergyDecayRate)
	require.Equal(t, p.OpenedAt.Add(cfg.NaturalTTLMax), p.ScheduledCloseAt)
}

func TestResolveNaturalSpawn_ReusesNaturalFactoryStability(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	rnd := &recordingRandom{ints: []int{0, 10, 0, 0, 2}, floats: []float64{10, .1, 0}}
	resolveDueNatural(t, &state, rnd)
	require.Equal(t, domain.PortalUnstable, state.Portals[0].Stability)
	require.NotNil(t, state.Portals[0].InstabilityCollapseAt)
}

func TestResolveNaturalSpawn_ReusesNaturalFactoryCreatures(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 4, 2, 10, .1, .9))
	require.Equal(t, 4, state.Portals[0].CreaturesInitial)
}

func TestResolveNaturalSpawn_SchedulesNextDelayAfterSuccess(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 12, 10, .1, .9))
	require.False(t, state.NaturalSpawn.Paused)
	require.Equal(t, testutil.BaseTime.Add(13*time.Second), *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_NextScheduleUsesSpawnTick(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(5 * time.Second)
	_, err := domain.ResolveNaturalSpawn(&state, now, naturalSpawnRandom(0, 10, 0, 3, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.Equal(t, now, *state.NaturalSpawn.ScheduledAt)
}

func TestResolveNaturalSpawn_NextZeroDelayDoesNotSpawnTwice(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	_, err := domain.ResolveNaturalSpawn(&state, now, naturalSpawnRandom(0, 10, 0, 0, 10, .1, .9), config.Default())
	require.NoError(t, err)
	before := append([]domain.Portal(nil), state.Portals...)
	result, err := domain.ResolveNaturalSpawn(&state, now, nil, config.Default())
	require.NoError(t, err)
	require.False(t, result.Spawned)
	require.Equal(t, before, state.Portals)
}

func TestResolveNaturalSpawn_SeventhPortalPausesWithoutNextDelayDraw(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = openPortals(6)
	state.NextPortalID = 7
	rnd := &recordingRandom{ints: []int{0, 10, 0}, floats: []float64{10, .1, .9}}
	resolveDueNatural(t, &state, rnd)
	require.True(t, state.NaturalSpawn.Paused)
	require.Empty(t, rnd.ints)
}

func TestResolveNaturalSpawn_SixthPortalDrawsNextDelay(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	state.Portals = openPortals(5)
	state.NextPortalID = 6
	rnd := naturalSpawnRandom(0, 10, 0, 8, 10, .1, .9)
	resolveDueNatural(t, &state, rnd)
	require.Empty(t, rnd.ints)
	require.Equal(t, testutil.BaseTime.Add(9*time.Second), *state.NaturalSpawn.DueAt)
}

func TestResolveNaturalSpawn_DestinationDrawPrecedesFactoryDraws(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	rnd := naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9)
	resolveDueNatural(t, &state, rnd)
	require.Equal(t, "int:0:84", rnd.calls[0])
	require.Equal(t, "int:10:300", rnd.calls[1])
}

func TestResolveNaturalSpawn_FactoryDrawsPrecedeNextDelay(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	rnd := naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9)
	resolveDueNatural(t, &state, rnd)
	require.Equal(t, "int:0:20", rnd.calls[len(rnd.calls)-1])
}

func TestResolveNaturalSpawn_NoDueSpawnConsumesNoRandom(t *testing.T) {
	state := simulationState(testutil.BaseTime)
	rnd := &countingRandom{}
	_, err := domain.ResolveNaturalSpawn(&state, testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}

func TestResolveNaturalSpawn_SuccessDoesNotMutatePlanes(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	before := append([]domain.Plane(nil), state.Planes...)
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, before, state.Planes)
}

func TestResolveNaturalSpawn_SuccessDoesNotMutateObserversOrLab(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	beforeObservers := append([]domain.Observer(nil), state.Observers...)
	beforeLab := state.Lab
	resolveDueNatural(t, &state, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9))
	require.Equal(t, beforeObservers, state.Observers)
	require.Equal(t, beforeLab, state.Lab)
}
