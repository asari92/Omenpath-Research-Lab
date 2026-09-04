package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func attentionPortal(id int64, slot int, ttl time.Duration) domain.Portal {
	p := testutil.NewPortalBuilder().Slot(slot).TTL(ttl).Build()
	p.ID = id
	p.Name = "attention"
	p.DestinationPlaneID = id
	return p
}

func terminalAttentionPortal(id int64, slot int, status domain.PortalStatus) domain.Portal {
	p := attentionPortal(id, slot, time.Minute)
	closedAt := testutil.BaseTime
	p.Status = status
	p.ClosedAt = &closedAt
	if status == domain.PortalStatusClosed {
		p.TerminationReason = domain.TerminationManualClose
	} else {
		p.TerminationReason = domain.TerminationEnergyDepleted
	}
	return p
}

func TestNeedsAttentionPortalIndex_ReturnsNoneWithoutOpenPortals(t *testing.T) {
	index, ok, err := domain.NeedsAttentionPortalIndex(nil, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, index)
}

func TestNeedsAttentionPortalIndex_IgnoresClosedPortal(t *testing.T) {
	open := attentionPortal(2, 2, time.Minute)
	portals := []domain.Portal{terminalAttentionPortal(1, 1, domain.PortalStatusClosed), open}
	index, ok, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_IgnoresCollapsedPortal(t *testing.T) {
	open := attentionPortal(2, 2, time.Minute)
	portals := []domain.Portal{terminalAttentionPortal(1, 1, domain.PortalStatusCollapsed), open}
	index, ok, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_SelectsHighestRiskScore(t *testing.T) {
	portals := []domain.Portal{attentionPortal(1, 1, time.Minute), attentionPortal(2, 2, 10*time.Second)}
	index, ok, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_RiskScoreBeatsInstabilityTieBreaker(t *testing.T) {
	stable := attentionPortal(1, 1, 5*time.Second)
	unstable := attentionPortal(2, 2, time.Minute)
	unstable.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(30 * time.Second)
	unstable.InstabilityCollapseAt = &hidden
	index, _, err := domain.NeedsAttentionPortalIndex([]domain.Portal{unstable, stable}, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_ExactRiskTiePrefersUnstable(t *testing.T) {
	stable := attentionPortal(2, 1, 36*time.Second)
	unstable := attentionPortal(1, 2, time.Minute)
	unstable.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(30 * time.Second)
	unstable.InstabilityCollapseAt = &hidden
	portals := []domain.Portal{stable, unstable}
	require.Equal(t, stable.RiskScore(testutil.BaseTime, config.Default()), unstable.RiskScore(testutil.BaseTime, config.Default()))
	index, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_ExactRiskAndStabilityTiePrefersLowerLifetime(t *testing.T) {
	portals := []domain.Portal{attentionPortal(1, 1, 60*time.Second), attentionPortal(2, 2, 50*time.Second)}
	index, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_LifetimeTiePrefersOlderOpening(t *testing.T) {
	older := attentionPortal(2, 1, time.Minute)
	newer := attentionPortal(1, 2, time.Minute)
	newer.OpenedAt = testutil.BaseTime.Add(time.Second)
	newer.CreatedAt = newer.OpenedAt
	newer.EnergyBaseAt = newer.OpenedAt
	newer.ScheduledCloseAt = older.ScheduledCloseAt
	index, _, err := domain.NeedsAttentionPortalIndex([]domain.Portal{newer, older}, testutil.BaseTime.Add(2*time.Second), config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_CompleteTiePrefersLowerPortalID(t *testing.T) {
	higher := attentionPortal(2, 1, time.Minute)
	lower := attentionPortal(1, 2, time.Minute)
	index, _, err := domain.NeedsAttentionPortalIndex([]domain.Portal{higher, lower}, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_ReturnsOriginalSliceIndex(t *testing.T) {
	portals := []domain.Portal{attentionPortal(3, 1, time.Minute), attentionPortal(1, 2, 5*time.Second), attentionPortal(2, 3, 30*time.Second)}
	index, ok, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_DoesNotReorderInput(t *testing.T) {
	portals := []domain.Portal{attentionPortal(3, 1, time.Minute), attentionPortal(1, 2, 5*time.Second), attentionPortal(2, 3, 30*time.Second)}
	beforeIDs := []int64{portals[0].ID, portals[1].ID, portals[2].ID}
	_, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, beforeIDs, []int64{portals[0].ID, portals[1].ID, portals[2].ID})
}

func TestNeedsAttentionPortalIndex_DoesNotMutatePortals(t *testing.T) {
	portals := []domain.Portal{attentionPortal(1, 1, time.Minute), attentionPortal(2, 2, 5*time.Second)}
	before := append([]domain.Portal(nil), portals...)
	_, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, before, portals)
}

func TestNeedsAttentionPortalIndex_UsesCurrentTimeDerivedRisk(t *testing.T) {
	first := attentionPortal(1, 1, 5*time.Minute)
	second := attentionPortal(2, 2, 5*time.Minute)
	first.EnergyDecayRate = 1
	second.EnergyDecayRate = 1
	second.EnergyBaseAt = testutil.BaseTime.Add(10 * time.Second)
	portals := []domain.Portal{first, second}
	index, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime.Add(80*time.Second), config.Default())
	require.NoError(t, err)
	require.Equal(t, 0, index)
	require.Greater(t, first.RiskScore(testutil.BaseTime.Add(80*time.Second), config.Default()), second.RiskScore(testutil.BaseTime.Add(80*time.Second), config.Default()))
}

func TestNeedsAttentionPortalIndex_HiddenTimestampDoesNotAffectScore(t *testing.T) {
	p1 := attentionPortal(2, 1, time.Minute)
	p2 := attentionPortal(1, 2, time.Minute)
	p1.Stability, p2.Stability = domain.PortalUnstable, domain.PortalUnstable
	h1, h2 := testutil.BaseTime.Add(5*time.Second), testutil.BaseTime.Add(50*time.Second)
	p1.InstabilityCollapseAt, p2.InstabilityCollapseAt = &h1, &h2
	index, _, err := domain.NeedsAttentionPortalIndex([]domain.Portal{p1, p2}, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.Equal(t, 1, index)
}

func TestNeedsAttentionPortalIndex_RejectsInvalidRiskConfig(t *testing.T) {
	cfg := config.Default()
	cfg.RiskSafeHorizon = 0
	_, _, err := domain.NeedsAttentionPortalIndex([]domain.Portal{attentionPortal(1, 1, time.Minute)}, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestNeedsAttentionPortalIndex_RejectionIsAtomic(t *testing.T) {
	cfg := config.Default()
	cfg.RiskSafeHorizon = -time.Second
	portals := []domain.Portal{attentionPortal(1, 1, time.Minute)}
	before := append([]domain.Portal(nil), portals...)
	_, _, err := domain.NeedsAttentionPortalIndex(portals, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
	require.Equal(t, before, portals)
}
