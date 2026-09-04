package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func tickNaturalClosePortal(id int64, slot int, after time.Duration) domain.Portal {
	p := testutil.NewPortalBuilder().Slot(slot).Energy(100).Decay(.1).TTL(after).Build()
	p.ID = id
	p.Name = "tick portal"
	p.DestinationPlaneID = id
	return p
}

func tickEnergyCollapsePortal(id int64, slot int, after time.Duration) domain.Portal {
	p := tickNaturalClosePortal(id, slot, time.Minute)
	p.EnergyBase = after.Seconds()
	p.EnergyDecayRate = 1
	return p
}

func tickInstabilityCollapsePortal(id int64, slot int, after time.Duration) domain.Portal {
	p := tickNaturalClosePortal(id, slot, time.Minute)
	p.Stability = domain.PortalUnstable
	at := testutil.BaseTime.Add(after)
	p.InstabilityCollapseAt = &at
	return p
}

func portalTickState(portals ...domain.Portal) domain.SimulationState {
	state := simulationState(testutil.BaseTime)
	state.Portals = portals
	state.NextPortalID = int64(len(portals) + 1)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 20*time.Second)
	return state
}

func TestSimulationTick_ResolvesNaturalClose(t *testing.T) {
	state := portalTickState(tickNaturalClosePortal(1, 1, 2*time.Second))
	result, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.True(t, result.Changed)
	require.Equal(t, domain.PortalStatusClosed, state.Portals[0].Status)
	require.Equal(t, domain.TerminationNaturalClose, state.Portals[0].TerminationReason)
}

func TestSimulationTick_ResolvesEnergyCollapse(t *testing.T) {
	state := portalTickState(tickEnergyCollapsePortal(1, 1, 2*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusCollapsed, state.Portals[0].Status)
	require.Equal(t, domain.TerminationEnergyDepleted, state.Portals[0].TerminationReason)
}

func TestSimulationTick_ResolvesInstabilityCollapse(t *testing.T) {
	state := portalTickState(tickInstabilityCollapsePortal(1, 1, 2*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusCollapsed, state.Portals[0].Status)
	require.Equal(t, domain.TerminationInstability, state.Portals[0].TerminationReason)
}

func TestSimulationTick_UsesSemanticPortalClosedAt(t *testing.T) {
	state := portalTickState(tickEnergyCollapsePortal(1, 1, 2*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.NotNil(t, state.Portals[0].ClosedAt)
	require.Equal(t, testutil.BaseTime.Add(2*time.Second), *state.Portals[0].ClosedAt)
	require.Equal(t, *state.Portals[0].ClosedAt, state.Lab.EnergyBaseAt)
}

func TestSimulationTick_NaturalCloseDoesNotActivateOverride(t *testing.T) {
	state := portalTickState(tickNaturalClosePortal(1, 1, 2*time.Second))
	before := state.Lab
	_, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Lab)
}

func TestSimulationTick_EnergyCollapseResetsLabEnergy(t *testing.T) {
	state := portalTickState(tickEnergyCollapsePortal(1, 1, 2*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Zero(t, state.Lab.EnergyBase)
}

func TestSimulationTick_InstabilityCollapseResetsLabEnergy(t *testing.T) {
	state := portalTickState(tickInstabilityCollapsePortal(1, 1, 2*time.Second))
	_, err := state.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Zero(t, state.Lab.EnergyBase)
}

func TestSimulationTick_MultipleCollapsesApplyChronologically(t *testing.T) {
	state := portalTickState(
		tickEnergyCollapsePortal(1, 1, 2*time.Second),
		tickEnergyCollapsePortal(2, 2, 4*time.Second),
	)
	result, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(4*time.Second), state.Lab.EnergyBaseAt)
	require.Equal(t, 1, result.LabEnergy)
}

func TestSimulationTick_MultipleCollapsesIgnorePortalSliceOrder(t *testing.T) {
	state := portalTickState(
		tickEnergyCollapsePortal(2, 2, 4*time.Second),
		tickEnergyCollapsePortal(1, 1, 2*time.Second),
	)
	result, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(4*time.Second), state.Lab.EnergyBaseAt)
	require.Equal(t, 1, result.LabEnergy)
	require.Equal(t, []int64{2, 1}, []int64{state.Portals[0].ID, state.Portals[1].ID})
}

func TestSimulationTick_EqualCollapseTimesUsePortalIDOrder(t *testing.T) {
	forward := portalTickState(
		tickEnergyCollapsePortal(1, 1, 2*time.Second),
		tickInstabilityCollapsePortal(2, 2, 2*time.Second),
	)
	reverse := portalTickState(forward.Portals[1], forward.Portals[0])
	_, err := forward.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	_, err = reverse.ResolveTick(testutil.BaseTime.Add(2*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, forward.Lab, reverse.Lab)
}

func TestSimulationTick_AlreadyTerminalPortalIsIdempotent(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, time.Minute)
	closedAt := testutil.BaseTime
	p.Status = domain.PortalStatusClosed
	p.TerminationReason = domain.TerminationManualClose
	p.ClosedAt = &closedAt
	state := portalTickState(p)
	before := state.Portals[0]
	_, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Portals[0])
}

func TestSimulationTick_TerminalPortalReleasesSlotBeforeNaturalStage(t *testing.T) {
	state := portalTickState(openPortals(6)...)
	closing := tickNaturalClosePortal(7, 7, time.Second)
	state.Portals = append(state.Portals, closing)
	state.NextPortalID = 8
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, time.Second)
	rnd := naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9)
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), rnd, config.Default())
	require.NoError(t, err)
	require.True(t, result.Spawned)
	require.Equal(t, 7, state.Portals[7].SlotIndex)
}

func TestSimulationTick_DerivedPortalEnergyIsNotPersisted(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, time.Minute)
	state := portalTickState(p)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, p.EnergyBase, state.Portals[0].EnergyBase)
	require.Equal(t, p.EnergyBaseAt, state.Portals[0].EnergyBaseAt)
}

func TestSimulationTick_DerivedCreaturesAreNotPersisted(t *testing.T) {
	p := tickNaturalClosePortal(1, 1, time.Minute)
	p.CreaturesInitial = 5
	state := portalTickState(p)
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, 5, state.Portals[0].CreaturesInitial)
}

func TestSimulationTick_DerivedLabEnergyIsNotRebased(t *testing.T) {
	state := portalTickState(tickNaturalClosePortal(1, 1, time.Minute))
	state.Lab.EnergyBase = 20
	before := state.Lab
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, state.Lab)
}

func TestSimulationTick_ResultReportsCurrentLabEnergy(t *testing.T) {
	state := portalTickState(tickNaturalClosePortal(1, 1, time.Minute))
	state.Lab.EnergyBase = 20
	result, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.NoError(t, err)
	require.Equal(t, 25, result.LabEnergy)
}

func TestSimulationTick_PortalStageFailureIsAtomic(t *testing.T) {
	state := portalTickState(tickEnergyCollapsePortal(1, 1, 2*time.Second))
	state.Lab.EnergyBaseAt = testutil.BaseTime.Add(3 * time.Second)
	before := state
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), nil, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, before, state)
}

func TestSimulationTick_PortalStageFailureConsumesNoRandom(t *testing.T) {
	state := portalTickState(tickEnergyCollapsePortal(1, 1, 2*time.Second))
	state.Lab.EnergyBaseAt = testutil.BaseTime.Add(3 * time.Second)
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, time.Second)
	rnd := &countingRandom{}
	_, err := state.ResolveTick(testutil.BaseTime.Add(5*time.Second), rnd, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Zero(t, rnd.intCalls)
	require.Zero(t, rnd.floatCalls)
}
