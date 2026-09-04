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
			Observers:    domain.NewObserverRoster(10, now.Add(-time.Hour)),
			NextPortalID: 2,
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
}

func TestBuildStateSnapshot_DoesNotExposeHiddenFields(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	hidden := testutil.BaseTime.Add(time.Minute)
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

func TestBuildPortalDetails_TerminalHasNullRisk(t *testing.T) {
	state := dtoSnapshot(testutil.BaseTime)
	closed := testutil.BaseTime
	state.Simulation.Portals[0].Status = domain.PortalStatusClosed
	state.Simulation.Portals[0].TerminationReason = domain.TerminationNaturalClose
	state.Simulation.Portals[0].ClosedAt = &closed
	got, err := BuildPortalDetails(state, 1, nil, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Nil(t, got.RiskLevel)
}

func TestBuildStateSnapshot_SlotsNeverContainRecommendation(t *testing.T) {
	got, err := BuildStateSnapshot(dtoSnapshot(testutil.BaseTime), testutil.BaseTime, config.Default())
	require.NoError(t, err)
	payload, err := json.Marshal(got)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(payload), "recommendation"))
}
