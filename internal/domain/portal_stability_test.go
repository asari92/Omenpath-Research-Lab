package domain_test

// Stage 2, substages 2.5–2.6 (04_STAGE_02_PORTAL_CORE_TDD.md):
// stability semantics, hidden instability collapse and the Stabilize
// primitive. No Lab Energy coupling here (stage boundary, plan §3).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// PORTAL-008 / STABILITY-003 (Final Spec §10): a hidden instability
// timestamp reached while OPEN+UNSTABLE collapses the portal.
func TestPortal_UnstableCollapsesAtHiddenTime(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(23 * time.Second)).
		Energy(100).Decay(0.1). // depletion far away
		TTL(60 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)

	// One nanosecond before the hidden moment: still OPEN.
	clk.Advance(23*time.Second - 1*time.Nanosecond)
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, domain.PortalStatusOpen, p.Status)

	// At the hidden moment: COLLAPSED / INSTABILITY.
	clk.Advance(1 * time.Nanosecond)
	changed, err = p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, p.Status)
	require.Equal(t, domain.TerminationInstability, p.TerminationReason)
	require.NotNil(t, p.ClosedAt)
	require.Equal(t, testutil.BaseTime.Add(23*time.Second), *p.ClosedAt)
}

// Stage 2 plan §12: natural close scheduled before the hidden collapse.
func TestPortal_NaturalCloseWinsBeforeHiddenCollapse(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(23 * time.Second)).
		Energy(100).Decay(0.1).
		TTL(10 * time.Second). // close at +10s, hidden at +23s
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(10 * time.Second)

	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *p.ClosedAt)
}

// Stage 2 plan Rule C: if the hidden timestamp exactly equals the natural
// close time, Natural Close wins the tie.
func TestPortal_NaturalCloseTiesHiddenCollapse(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(10 * time.Second)).
		Energy(100).Decay(0.1).
		TTL(10 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(10 * time.Second)

	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
}

// Stabilized portals keep the earliest-wins semantics without the hidden
// candidate: STABILITY-005/006 end-to-end through the lifecycle.
func TestPortal_StabilizedPortalLosesInstabilityCandidate(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(23 * time.Second)).
		Energy(50).Decay(0.1). // depletion far away; ≤85 so Stabilize is legal
		TTL(60 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	require.NoError(t, p.Stabilize(clk.Now(), config.Default()))

	// Well past the former hidden moment: natural close still decides.
	clk.Advance(59 * time.Second)
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, domain.PortalStatusOpen, p.Status)

	clk.Advance(1 * time.Second)
	changed, err = p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
}

// STABILITY-005/006, ENERGY-006/007/008 (Final Spec §11): happy path.
func TestPortal_StabilizeConvertsUnstableToStable(t *testing.T) {
	hidden := testutil.BaseTime.Add(30 * time.Second)
	p := testutil.NewPortalBuilder().
		Unstable(hidden).
		Energy(50).Decay(0.5).
		TTL(60 * time.Second).
		Build()

	now := testutil.BaseTime // current energy = 50
	require.NoError(t, p.Stabilize(now, config.Default()))

	require.Equal(t, domain.PortalStable, p.Stability)
	require.Nil(t, p.InstabilityCollapseAt)
	require.InDelta(t, 65.0, p.EnergyBase, 1e-9) // 50 + 15
	require.Equal(t, now, p.EnergyBaseAt)        // new baseline anchored at now
	require.InDelta(t, 0.5, p.EnergyDecayRate, 1e-9)
	require.Equal(t, now, p.UpdatedAt)
	require.Equal(t, 65.0, p.CurrentEnergy(now)) // derived confirms re-baseline
}

// Stage 2 plan §14 regression: Stabilize must add +15 to the CURRENT
// derived energy, not to the stale baseline (80−5=75 → 90, not 95).
func TestPortal_StabilizeUsesCurrentEnergyNotBaseline(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(50 * time.Second)).
		Energy(80).Decay(0.5).
		TTL(60 * time.Second).
		Build()

	now := testutil.BaseTime.Add(10 * time.Second) // current = 80 − 5 = 75
	require.NoError(t, p.Stabilize(now, config.Default()))

	require.InDelta(t, 90.0, p.EnergyBase, 1e-9) // 75 + 15, NOT 80 + 15
	require.Equal(t, now, p.EnergyBaseAt)
	require.InDelta(t, 90.0, p.CurrentEnergy(now), 1e-9)
}

// ENERGY-010 (Final Spec §11): exactly 85% is allowed and yields 100%.
func TestPortal_StabilizeAllowsExactly85Percent(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(50 * time.Second)).
		Energy(85).Decay(0.5).
		TTL(60 * time.Second).
		Build()

	now := testutil.BaseTime // current = 85 exactly
	require.NoError(t, p.Stabilize(now, config.Default()))
	require.InDelta(t, 100.0, p.EnergyBase, 1e-9)
}

// ENERGY-009 (Final Spec §11): current energy above 85% is rejected and
// the portal remains entirely unchanged (mutation safety, plan §22).
func TestPortal_StabilizeRejectsEnergyAbove85(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(50 * time.Second)).
		Energy(86).Decay(0.5).
		TTL(60 * time.Second).
		Build()
	snapshot := p

	err := p.Stabilize(testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalOverchargeRisk)
	require.Equal(t, snapshot, p)
}

// The check uses the derived current energy: a portal whose baseline is
// above 85 but has decayed below it must be stabilizable.
func TestPortal_StabilizeAllowedWhenCurrentBelowThreshold(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(50 * time.Second)).
		Energy(90).Decay(1.0).
		TTL(60 * time.Second).
		Build()

	now := testutil.BaseTime.Add(10 * time.Second) // current = 90 − 10 = 80
	require.NoError(t, p.Stabilize(now, config.Default()))
	require.InDelta(t, 95.0, p.EnergyBase, 1e-9)
}

// STABILITY-007 (Final Spec §11): STABLE portals cannot be stabilized.
func TestPortal_StabilizeRejectsStablePortal(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).Build()
	snapshot := p

	err := p.Stabilize(testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
	require.Equal(t, snapshot, p)
}

// Stage 2 plan §14: terminal portals reject Stabilize.
func TestPortal_StabilizeRejectsTerminalPortal(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Unstable(testutil.BaseTime.Add(50 * time.Second)).
			Energy(50).Decay(0.5).TTL(10 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusClosed, p.Status)
		snapshot := p

		require.ErrorIs(t, p.Stabilize(clk.Now(), config.Default()), domain.ErrPortalNotOpen)
		require.Equal(t, snapshot, p)
	})

	t.Run("collapsed by instability", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Unstable(testutil.BaseTime.Add(6 * time.Second)).
			Energy(50).Decay(0.5).TTL(60 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(6 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)
		snapshot := p

		require.ErrorIs(t, p.Stabilize(clk.Now(), config.Default()), domain.ErrPortalNotOpen)
		require.Equal(t, snapshot, p)
	})
}
