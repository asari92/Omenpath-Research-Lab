package domain_test

// Corrective pass after the independent Stage 0–2 audit: terminal derived
// state. Once a portal is CLOSED or COLLAPSED it must stop behaving like an
// active one — derived values freeze at the ClosedAt moment (PORTAL-009
// extended to derived state; PORTAL-004 distinct outcomes):
//
//	ScheduledRemaining  → 0
//	CurrentEnergy       → frozen at ClosedAt
//	CreaturesInside     → frozen at ClosedAt
//	RiskLevel           → still absent
//
// Plus a regression lock for the exact tie between energy depletion and the
// hidden instability moment: ENERGY_DEPLETED wins (Stage 2 plan Rule D).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// Closed at +6s with creatures still inside (4 − 3 = 1): the count freezes
// at 1 forever instead of draining to zero like an active portal would.
func TestPortal_CreaturesFreezeAtManualClose(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().Energy(100).Decay(0.1).
		TTL(120 * time.Second).
		Creatures(4).
		Build()

	now := testutil.BaseTime.Add(6 * time.Second)
	require.Equal(t, 1, p.CreaturesInside(now, config.Default()))
	require.NoError(t, p.Close(now, true, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, p.Status)

	require.Equal(t, 1, p.CreaturesInside(now, config.Default()))
	require.Equal(t, 1, p.CreaturesInside(testutil.BaseTime.Add(60*time.Second), config.Default()),
		"terminal portal creatures must stay frozen at the close moment")
	require.Equal(t, 1, p.CreaturesInside(testutil.BaseTime.Add(1000*time.Second), config.Default()))
}

// Collapsed by instability at +6s: creatures freeze at the collapse moment
// (1), and risk remains absent at any later time.
func TestPortal_CreaturesFreezeAtCollapse(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(6 * time.Second)).
		Energy(100).Decay(0.1).
		TTL(120 * time.Second).
		Creatures(4).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(6 * time.Second)
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, p.Status)

	require.Equal(t, 1, p.CreaturesInside(clk.Now(), config.Default()))
	require.Equal(t, 1, p.CreaturesInside(testutil.BaseTime.Add(600*time.Second), config.Default()),
		"terminal portal creatures must stay frozen at the collapse moment")

	_, ok := p.RiskLevel(testutil.BaseTime.Add(600*time.Second), config.Default())
	require.False(t, ok, "risk must stay absent for a terminal portal")
}

// Manual close at +10s (energy 100, decay 0.5 → 95 at close): energy freezes
// at 95 instead of continuing to decay.
func TestPortal_EnergyFreezesAtManualClose(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().Energy(100).Decay(0.5).
		TTL(120 * time.Second).
		Build()

	now := testutil.BaseTime.Add(10 * time.Second)
	require.NoError(t, p.Close(now, false, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, p.Status)

	require.InDelta(t, 95.0, p.CurrentEnergy(now), 1e-9)
	require.InDelta(t, 95.0, p.CurrentEnergy(testutil.BaseTime.Add(100*time.Second)), 1e-9,
		"terminal portal energy must stay frozen at the close moment")
	require.InDelta(t, 95.0, p.CurrentEnergy(testutil.BaseTime.Add(1000*time.Second)), 1e-9)
}

// Collapsed portals freeze their energy at the collapse moment too. An
// instability collapse keeps a non-zero frozen value; an energy-depleted
// collapse freezes at exactly zero.
func TestPortal_EnergyFreezesAtCollapse(t *testing.T) {
	t.Run("instability collapse freezes at non-zero energy", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Unstable(testutil.BaseTime.Add(10 * time.Second)).
			Energy(100).Decay(0.5).
			TTL(120 * time.Second).
			Build()

		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		changed, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)

		require.InDelta(t, 95.0, p.CurrentEnergy(clk.Now()), 1e-9)
		require.InDelta(t, 95.0, p.CurrentEnergy(testutil.BaseTime.Add(1000*time.Second)), 1e-9,
			"terminal portal energy must stay frozen at the collapse moment")

		_, ok := p.RiskLevel(testutil.BaseTime.Add(1000*time.Second), config.Default())
		require.False(t, ok, "risk must stay absent for a terminal portal")
	})

	t.Run("energy depletion collapses frozen at zero", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(10).Decay(1.0). // depletion exactly at +10s
			TTL(120 * time.Second).
			Build()

		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		changed, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.True(t, changed)
		require.Equal(t, domain.TerminationEnergyDepleted, p.TerminationReason)

		require.InDelta(t, 0.0, p.CurrentEnergy(testutil.BaseTime.Add(1000*time.Second)), 1e-9)
	})
}

// A terminal portal has no countdown left, even when the scheduled close
// lies in the future at query time.
func TestPortal_ScheduledRemainingIsZeroWhenTerminal(t *testing.T) {
	t.Run("manual close before scheduled close", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(100).Decay(0.1).
			TTL(60 * time.Second).
			Build()
		require.NoError(t, p.Close(testutil.BaseTime.Add(5*time.Second), false, config.Default()))

		require.Equal(t, time.Duration(0), p.ScheduledRemaining(testutil.BaseTime.Add(5*time.Second)))
		require.Equal(t, time.Duration(0), p.ScheduledRemaining(testutil.BaseTime.Add(30*time.Second)),
			"terminal portal must not report a remaining countdown")
	})

	t.Run("natural close", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(100).Decay(0.1).
			TTL(10 * time.Second).
			Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)

		require.Equal(t, time.Duration(0), p.ScheduledRemaining(clk.Now()))
	})

	t.Run("energy collapse before scheduled close", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(10).Decay(1.0). // depletion at +10s
			TTL(60 * time.Second).
			Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)

		require.Equal(t, time.Duration(0), p.ScheduledRemaining(testutil.BaseTime.Add(20*time.Second)))
	})

	t.Run("instability collapse before scheduled close", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Unstable(testutil.BaseTime.Add(6 * time.Second)).
			Energy(100).Decay(0.1).
			TTL(60 * time.Second).
			Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(6 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)

		require.Equal(t, time.Duration(0), p.ScheduledRemaining(clk.Now()))
	})
}

// Regression lock for the fixed tie semantics (Stage 2 plan Rule D): when
// energy depletion and the hidden instability moment coincide exactly and
// both are strictly before natural close, ENERGY_DEPLETED wins. Such a tie
// IS reachable from the valid factory (both moments may lie strictly inside
// the TTL window), so the choice must stay deterministic and documented.
func TestPortal_EnergyDepletionWinsInstabilityTie(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(100 * time.Second)). // hidden == depletion
		Energy(50).Decay(0.5).                              // depletion at +100s
		TTL(200 * time.Second).                             // close at +200s
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(100 * time.Second)

	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, p.Status)
	require.Equal(t, domain.TerminationEnergyDepleted, p.TerminationReason,
		"exact tie between energy depletion and instability keeps ENERGY_DEPLETED")
	require.NotNil(t, p.ClosedAt)
	require.Equal(t, testutil.BaseTime.Add(100*time.Second), *p.ClosedAt)
}
