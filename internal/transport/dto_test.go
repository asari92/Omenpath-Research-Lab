package transport

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

func dtoSnapshot(now time.Time) persistence.Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: "Plane", Aliases: []string{}, CatalogTier: "core"}
	}
	portal := testutil.NewPortalBuilder().Creatures(2).Build()
	portal.EnergyBaseAt = now.Add(-10 * time.Second)
	portal.OpenedAt = now.Add(-10 * time.Second)
	portal.CreatedAt = portal.OpenedAt
	portal.UpdatedAt = portal.OpenedAt
	portal.ScheduledCloseAt = now.Add(50 * time.Second)
	return persistence.Snapshot{
		Simulation: domain.SimulationState{
			Lab:          domain.LabState{EnergyBase: 50, EnergyBaseAt: now.Add(-5 * time.Second)},
			Portals:      []domain.Portal{portal},
			Planes:       planes,
			Observers:    domain.NewObserverRoster(config.Default().ObserverCount, now.Add(-time.Hour)),
			NextPortalID: 2,
			NaturalSpawn: domain.NaturalSpawnState{Paused: true},
		},
		App: domain.AppState{Mode: domain.ModeLive, TutorialStep: 9},
	}
}

func TestBuildStateSnapshot_AlwaysReturnsSevenFixedSlots(t *testing.T) {
	got, err := BuildStateSnapshot(dtoSnapshot(testutil.BaseTime), testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Len(t, got.Slots, 7)
	require.NotNil(t, got.Slots[0].Portal)
	for i := 1; i < 7; i++ {
		require.Equal(t, i+1, got.Slots[i].SlotIndex)
		require.Nil(t, got.Slots[i].Portal)
	}
}

func TestBuildStateSnapshot_DerivesCurrentValuesAtGeneratedAt(t *testing.T) {
	now := testutil.BaseTime
	got, err := BuildStateSnapshot(dtoSnapshot(now), now, config.Default())
	require.NoError(t, err)
	require.Equal(t, now, got.GeneratedAt)
	require.Equal(t, 55, got.Lab.CurrentEnergy)
	require.Equal(t, 95.0, got.Slots[0].Portal.Energy)
	require.Equal(t, int64(50), got.Slots[0].Portal.TimeRemainingSeconds)
	require.Equal(t, 0, got.Slots[0].Portal.CreaturesInside)
	require.Equal(t, 20, observerStatusTotal(got.Observers))
}

func observerStatusTotal(counts ObserverCountsDTO) int {
	return counts.Available + counts.Outbound + counts.Exploring + counts.WaitingReturn + counts.Returning + counts.Lost
}

func TestBuildStateSnapshot_PlaneObserverPresence(t *testing.T) {
	now := testutil.BaseTime
	state := dtoSnapshot(now)
	state.Simulation.Portals = nil
	planeID, otherPlaneID := int64(1), int64(2)
	state.Simulation.Observers = []domain.Observer{
		{ID: 1, Status: domain.ObserverExploring, CurrentPlaneID: &planeID},
		{ID: 2, Status: domain.ObserverWaitingReturn, CurrentPlaneID: &planeID},
		{ID: 3, Status: domain.ObserverReturning, CurrentPlaneID: &planeID},
		{ID: 4, Status: domain.ObserverOutbound},
		{ID: 5, Status: domain.ObserverAvailable},
		{ID: 6, Status: domain.ObserverLost},
		{ID: 7, Status: domain.ObserverExploring, CurrentPlaneID: &otherPlaneID},
	}

	view, err := BuildStateSnapshot(state, now, config.Default())
	require.NoError(t, err)
	require.Equal(t, 3, view.Planes[0].ObserversInPlane)
	require.Equal(t, 1, view.Planes[0].ObserversWaitingReturn)
	require.Equal(t, 1, view.Planes[1].ObserversInPlane)
	require.Equal(t, 0, view.Planes[1].ObserversWaitingReturn)
}

func TestBuildStateSnapshot_PlaneObserverPresenceRejectsUnknownPlane(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	state.Simulation.Portals = nil
	unknown := int64(999)
	state.Simulation.Observers = []domain.Observer{{
		ID: 1, Status: domain.ObserverExploring, CurrentPlaneID: &unknown,
	}}

	_, err := BuildStateSnapshot(state, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestBuildStateSnapshot_DoesNotExposeHiddenFields(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	hidden := testutil.BaseTime.Add(30 * time.Second)
	state.Simulation.Portals[0].Stability = domain.PortalUnstable
	state.Simulation.Portals[0].InstabilityCollapseAt = &hidden
	got, err := BuildStateSnapshot(state, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	payload, err := json.Marshal(got)
	require.NoError(t, err)
	text := string(payload)
	for _, forbidden := range []string{"decay", "instability_collapse", "risk_score", "energy_lifetime"} {
		require.NotContains(t, text, forbidden)
	}
}

func TestBuildPortalDetails_OpenIncludesRiskAndHistory(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	id := int64(1)
	history := []domain.Event{{
		ID: 1, EventType: domain.EventPortalOpened, PortalID: &id,
		Message: "opened", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime,
	}}
	got, err := BuildPortalDetails(state, 1, history, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.NotNil(t, got.RiskLevel)
	require.Equal(t, domain.RiskLow, *got.RiskLevel)
	require.Len(t, got.History, 1)
	require.Equal(t, int64(1), got.Destination.PlaneID)
}

func TestBuildPortalDetails_OpenIncludesRecommendation(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	got, err := BuildPortalDetails(state, 1, nil, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.NotNil(t, got.Recommendation)
	require.Equal(t, domain.RecommendationSendObserver, *got.Recommendation)
}

func TestBuildPortalDetails_TerminalHasNullRisk(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	closed := testutil.BaseTime
	state.Simulation.Portals[0].Status = domain.PortalStatusClosed
	state.Simulation.Portals[0].TerminationReason = domain.TerminationManualClose
	state.Simulation.Portals[0].ClosedAt = &closed
	state.Simulation.Portals[0].UpdatedAt = closed
	got, err := BuildPortalDetails(state, 1, nil, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Nil(t, got.RiskLevel)
	require.Nil(t, got.Recommendation)
}

func TestBuildStateSnapshot_SlotsNeverContainRecommendation(t *testing.T) {
	got, err := BuildStateSnapshot(dtoSnapshot(testutil.BaseTime), testutil.BaseTime, config.Default())
	require.NoError(t, err)
	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(payload), "recommendation"))
}

func TestTutorialSnapshot_ExposesStepPhaseTargetsAndExpectedAction(t *testing.T) {
	snapshot := dtoSnapshot(testutil.BaseTime)
	portalID, planeID, observerID := int64(1), int64(1), int64(2)
	snapshot.App = domain.AppState{
		Mode: domain.ModeTutorial, TutorialStep: 6, TutorialPhase: domain.TutorialPhaseRecallReady,
		TutorialPortalID: &portalID, TutorialPlaneID: &planeID, TutorialObserverID: &observerID,
	}
	got, err := BuildStateSnapshot(snapshot, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, domain.TutorialPhaseRecallReady, got.App.TutorialPhase)
	require.Equal(t, &portalID, got.App.TutorialPortalID)
	require.Equal(t, &planeID, got.App.TutorialPlaneID)
	require.Equal(t, &observerID, got.App.TutorialObserverID)
	require.Equal(t, domain.TutorialActionRecall, *got.App.ExpectedAction)
}

func TestTutorialSnapshot_DoesNotExposePreparedHiddenValues(t *testing.T) {
	snapshot := dtoSnapshot(testutil.BaseTime)
	snapshot.App = domain.AppState{Mode: domain.ModeTutorial, TutorialStep: 1, TutorialPortalID: int64DTOTestPointer(1)}
	got, err := BuildStateSnapshot(snapshot, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	payload, err := json.Marshal(got.App)
	require.NoError(t, err)
	for _, forbidden := range []string{"decay", "instability", "collapse_at", "prepared"} {
		require.NotContains(t, string(payload), forbidden)
	}
}

func int64DTOTestPointer(value int64) *int64 { return &value }

func TestQuickActionReasons_ArePairedWithAvailability(t *testing.T) {
	got, err := BuildStateSnapshot(dtoSnapshot(testutil.BaseTime), testutil.BaseTime, config.Default())
	require.NoError(t, err)
	actions := got.Slots[0].Portal.QuickActions
	require.False(t, actions.CanStabilize)
	require.NotNil(t, actions.StabilizeUnavailableReason)
	require.Equal(t, "PORTAL_ALREADY_STABLE", *actions.StabilizeUnavailableReason)
	require.True(t, actions.CanClose)
	require.Nil(t, actions.CloseUnavailableReason)
	require.False(t, actions.CanRecall)
	require.Equal(t, "NO_WAITING_OBSERVER", *actions.RecallUnavailableReason)
}
