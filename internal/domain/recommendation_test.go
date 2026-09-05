package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func recommendationState(now time.Time) domain.SimulationState {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: "Plane", Aliases: []string{}, CatalogTier: "core"}
	}
	portal := testutil.NewPortalBuilder().Build()
	portal.OpenedAt, portal.CreatedAt, portal.UpdatedAt, portal.EnergyBaseAt = now, now, now, now
	portal.ScheduledCloseAt = now.Add(time.Minute)
	return domain.SimulationState{
		Lab: domain.LabState{EnergyBase: 100, EnergyBaseAt: now}, Portals: []domain.Portal{portal},
		Planes: planes, Observers: domain.NewObserverRoster(config.Default().ObserverCount, now), NextPortalID: 2,
		NaturalSpawn: domain.NaturalSpawnState{Paused: true},
	}
}

func recommendation(t *testing.T, state domain.SimulationState, now time.Time) domain.Recommendation {
	t.Helper()
	got, ok, err := domain.RecommendationForPortal(state, 1, now, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	return got
}

func recWaitingObserver(state *domain.SimulationState, index int, now time.Time) {
	planeID := int64(1)
	started := now.Add(-time.Minute)
	state.Observers[index].Status = domain.ObserverWaitingReturn
	state.Observers[index].CurrentPlaneID = &planeID
	state.Observers[index].PhaseStartedAt = &started
	state.Observers[index].UpdatedAt = now
}

func recExploringObserver(state *domain.SimulationState, index int, now time.Time, remaining time.Duration) {
	planeID := int64(1)
	started := now.Add(-(config.Default().ResearchDuration - remaining))
	ends := now.Add(remaining)
	state.Observers[index].Status = domain.ObserverExploring
	state.Observers[index].CurrentPlaneID = &planeID
	state.Observers[index].PhaseStartedAt = &started
	state.Observers[index].PhaseEndsAt = &ends
	state.Observers[index].UpdatedAt = now
}

func TestRecommendation_TerminalHasNoValue(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	closedAt := now
	state.Portals[0].Status = domain.PortalStatusClosed
	state.Portals[0].TerminationReason = domain.TerminationManualClose
	state.Portals[0].ClosedAt = &closedAt
	got, ok, err := domain.RecommendationForPortal(state, 1, now, config.Default())
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, got)
}

func TestRecommendation_ExtractionBeforeSyncLeavesOpen(t *testing.T) {
	state := recommendationState(testutil.BaseTime)
	state.Portals[0].Kind = domain.PortalKindExtraction
	state.Portals[0].ObserverFlow = domain.PortalFlowInbound
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, testutil.BaseTime))
}

func TestRecommendation_ActiveSafeTransitLeavesOpen(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	portalID := int64(1)
	started, ends := now.Add(-time.Second), now.Add(10*time.Second)
	state.Observers[0].Status = domain.ObserverOutbound
	state.Observers[0].ActivePortalID = &portalID
	state.Observers[0].PhaseStartedAt, state.Observers[0].PhaseEndsAt = &started, &ends
	state.Observers[0].UpdatedAt = now
	state.Portals[0].ObserverFlow = domain.PortalFlowOutbound
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
}

func TestRecommendation_ActiveUnsafeTransitStabilizesOnlyWhenHelpful(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	portalID := int64(1)
	started, ends := now.Add(-time.Second), now.Add(10*time.Second)
	state.Observers[0].Status = domain.ObserverOutbound
	state.Observers[0].ActivePortalID = &portalID
	state.Observers[0].PhaseStartedAt, state.Observers[0].PhaseEndsAt = &started, &ends
	state.Observers[0].UpdatedAt = now
	state.Portals[0].ObserverFlow = domain.PortalFlowOutbound
	hidden := now.Add(30 * time.Second)
	state.Portals[0].Stability = domain.PortalUnstable
	state.Portals[0].InstabilityCollapseAt = &hidden
	state.Portals[0].EnergyBase = 1
	state.Portals[0].EnergyDecayRate = 1
	require.Equal(t, domain.RecommendationStabilize, recommendation(t, state, now))
	state.Portals[0].ScheduledCloseAt = now.Add(10 * time.Second)
	hidden = now.Add(9 * time.Second)
	state.Portals[0].InstabilityCollapseAt = &hidden
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
}

func TestRecommendation_ActiveTransitNeverCloses(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	portalID := int64(1)
	started, ends := now.Add(-time.Second), now.Add(10*time.Second)
	state.Observers[0].Status = domain.ObserverReturning
	planeID := int64(1)
	state.Observers[0].CurrentPlaneID = &planeID
	state.Observers[0].ActivePortalID = &portalID
	state.Observers[0].PhaseStartedAt, state.Observers[0].PhaseEndsAt = &started, &ends
	state.Observers[0].UpdatedAt = now
	state.Portals[0].ObserverFlow = domain.PortalFlowInbound
	state.Portals[0].ScheduledCloseAt = now.Add(5 * time.Second)
	state.Lab.EnergyBase = 100
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
}

func TestRecommendation_WaitingObserverRecalls(t *testing.T) {
	state := recommendationState(testutil.BaseTime)
	recWaitingObserver(&state, 0, testutil.BaseTime)
	require.Equal(t, domain.RecommendationRecallObserver, recommendation(t, state, testutil.BaseTime))
}

func TestRecommendation_WaitingObserverWaitsForSafeCorridor(t *testing.T) {
	state := recommendationState(testutil.BaseTime)
	recWaitingObserver(&state, 0, testutil.BaseTime)
	state.Portals[0].CreaturesInitial = 2
	require.Equal(t, domain.RecommendationWaitForCorridor, recommendation(t, state, testutil.BaseTime))
}

func TestRecommendation_WaitingObserverStabilizesForSafeReturn(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	recWaitingObserver(&state, 0, now)
	hidden := now.Add(30 * time.Second)
	state.Portals[0].Stability = domain.PortalUnstable
	state.Portals[0].InstabilityCollapseAt = &hidden
	state.Portals[0].EnergyBase = 5
	state.Portals[0].EnergyDecayRate = 1
	require.Equal(t, domain.RecommendationStabilize, recommendation(t, state, now))
}

func TestRecommendation_ExploringObserverPreservesInboundPath(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	recExploringObserver(&state, 0, now, 10*time.Second)
	state.Portals[0].ObserverFlow = domain.PortalFlowInbound
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
	state.Portals[0].Stability = domain.PortalUnstable
	hidden := now.Add(40 * time.Second)
	state.Portals[0].InstabilityCollapseAt = &hidden
	state.Portals[0].EnergyBase = 15
	state.Portals[0].EnergyDecayRate = 1
	require.Equal(t, domain.RecommendationStabilize, recommendation(t, state, now))
}

func TestRecommendation_OutboundPathIsNotPreservedForRecall(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	recExploringObserver(&state, 0, now, 10*time.Second)
	state.Portals[0].ObserverFlow = domain.PortalFlowOutbound
	state.Portals[0].Stability = domain.PortalUnstable
	hidden := now.Add(40 * time.Second)
	state.Portals[0].InstabilityCollapseAt = &hidden
	state.Portals[0].EnergyBase = 15
	state.Portals[0].EnergyDecayRate = 1
	require.NotEqual(t, domain.RecommendationStabilize, recommendation(t, state, now))
}

func TestRecommendation_UnexploredPlaneSendsAvailableObserver(t *testing.T) {
	state := recommendationState(testutil.BaseTime)
	require.Equal(t, domain.RecommendationSendObserver, recommendation(t, state, testutil.BaseTime))
}

func TestRecommendation_DoesNotSendToExploredOrOccupiedPlane(t *testing.T) {
	now := testutil.BaseTime
	for _, mutate := range []func(*domain.SimulationState){
		func(state *domain.SimulationState) {
			explored := now
			state.Planes[0].Explored, state.Planes[0].ExploredAt = true, &explored
		},
		func(state *domain.SimulationState) { recWaitingObserver(state, 0, now) },
	} {
		state := recommendationState(now)
		mutate(&state)
		require.NotEqual(t, domain.RecommendationSendObserver, recommendation(t, state, now))
	}
}

func TestRecommendation_OtherOutboundToSamePlaneBlocksDuplicateSend(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	second := state.Portals[0]
	second.ID, second.Name, second.SlotIndex = 2, "Omenpath #0002", 2
	state.Portals = append(state.Portals, second)
	state.NextPortalID = 3
	portalID := int64(2)
	started, ends := now.Add(-time.Second), now.Add(10*time.Second)
	state.Observers[0].Status = domain.ObserverOutbound
	state.Observers[0].ActivePortalID = &portalID
	state.Observers[0].PhaseStartedAt, state.Observers[0].PhaseEndsAt = &started, &ends
	state.Observers[0].UpdatedAt = now
	state.Portals[1].ObserverFlow = domain.PortalFlowOutbound
	require.NotEqual(t, domain.RecommendationSendObserver, recommendation(t, state, now))
}

func TestRecommendation_ExactDeadlineTieIsUnsafe(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	state.Portals[0].ScheduledCloseAt = now.Add(config.Default().ObserverTransitMax)
	state.Lab.EnergyBase = 0
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
}

func TestRecommendation_DangerFallbackClosesWhenAffordable(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	state.Portals[0].ScheduledCloseAt = now.Add(10 * time.Second)
	for i := range state.Observers {
		state.Observers[i].Status = domain.ObserverLost
	}
	require.Equal(t, domain.RecommendationClose, recommendation(t, state, now))
}

func TestRecommendation_InsufficientLabEnergyLeavesOpen(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	state.Portals[0].ScheduledCloseAt = now.Add(10 * time.Second)
	state.Lab.EnergyBase = 0
	for i := range state.Observers {
		state.Observers[i].Status = domain.ObserverLost
	}
	require.Equal(t, domain.RecommendationLeaveOpen, recommendation(t, state, now))
}

func TestRecommendation_HypotheticalStabilizeDoesNotMutateState(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	hidden := now.Add(30 * time.Second)
	state.Portals[0].Stability = domain.PortalUnstable
	state.Portals[0].InstabilityCollapseAt = &hidden
	state.Portals[0].EnergyBase = 5
	state.Portals[0].EnergyDecayRate = 1
	want := state
	want.Portals = append([]domain.Portal(nil), state.Portals...)
	_, _, err := domain.RecommendationForPortal(state, 1, now, config.Default())
	require.NoError(t, err)
	require.Equal(t, want, state)
}

func TestRecommendation_DoesNotConsumeRandom(t *testing.T) {
	state := recommendationState(testutil.BaseTime)
	// No random dependency is accepted by the public function, so repeated calls
	// over the same inputs must return the same result.
	first := recommendation(t, state, testutil.BaseTime)
	second := recommendation(t, state, testutil.BaseTime)
	require.Equal(t, first, second)
}

func TestRecommendation_HiddenCollapseTimestampDoesNotAffectResult(t *testing.T) {
	now := testutil.BaseTime
	first := recommendationState(now)
	first.Portals[0].Stability = domain.PortalUnstable
	first.Portals[0].EnergyBase = 5
	first.Portals[0].EnergyDecayRate = 1
	early, late := now.Add(5*time.Second), now.Add(55*time.Second)
	first.Portals[0].InstabilityCollapseAt = &early
	second := first
	second.Portals = append([]domain.Portal(nil), first.Portals...)
	second.Portals[0].InstabilityCollapseAt = &late
	require.Equal(t, recommendation(t, first, now), recommendation(t, second, now))
}

func TestRecommendation_DoesNotRestrictDomainCommands(t *testing.T) {
	now := testutil.BaseTime
	state := recommendationState(now)
	explored := now
	state.Planes[0].Explored, state.Planes[0].ExploredAt = true, &explored
	require.Equal(t, domain.RecommendationClose, recommendation(t, state, now))
	_, err := domain.SendObserverWithLabEnergy(&state.Lab, &state.Portals[0], &state.Planes[0], state.Observers, now, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
}
