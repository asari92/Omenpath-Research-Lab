package domain_test

// The first RED cycle of the project established these lifecycle rules:
//
//	PORTAL-005  expired TTL → CLOSED + NATURAL_CLOSE
//	ENERGY-005  energy reaches 0 → COLLAPSED + ENERGY_DEPLETED
//	PORTAL-009  terminal portal can never return OPEN
//
// Energy derivation, stability, creatures, risk and slots have focused tests.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// PORTAL-005 (Final Spec §8): at TTL expiry OPEN → CLOSED with
// termination_reason = NATURAL_CLOSE. Before expiry the portal stays OPEN.
func TestPortal_NaturalCloseWhenTTLExpires(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().
		Energy(100).
		Decay(0.1). // depletion at +1000s, far beyond the 60s TTL
		TTL(60 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)

	// One second before expiry: still OPEN.
	clk.Advance(59 * time.Second)
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.False(t, changed, "portal must stay OPEN before scheduled close")
	require.Equal(t, domain.PortalStatusOpen, p.Status)

	// Exactly at expiry: CLOSED / NATURAL_CLOSE.
	clk.Advance(1 * time.Second)
	changed, err = p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
	require.NotNil(t, p.ClosedAt)
	// ClosedAt must store the semantic event time (§8), not a lazy tick time.
	require.Equal(t, testutil.BaseTime.Add(60*time.Second), *p.ClosedAt)
}

// ENERGY-005 / PORTAL-007 (Final Spec §9): energy reaching zero strictly
// before the natural close collapses the portal.
func TestPortal_CollapsesWhenEnergyReachesZeroBeforeNaturalClose(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().
		Energy(10).
		Decay(1.0). // depletion exactly at +10s
		TTL(60 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)

	// Just before depletion: still OPEN.
	clk.Advance(10*time.Second - 1*time.Nanosecond)
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, domain.PortalStatusOpen, p.Status)

	// At depletion: COLLAPSED / ENERGY_DEPLETED.
	clk.Advance(1 * time.Nanosecond)
	changed, err = p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, p.Status)
	require.Equal(t, domain.TerminationEnergyDepleted, p.TerminationReason)
	require.NotNil(t, p.ClosedAt)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *p.ClosedAt)
}

// PORTAL-009 (Final Spec §4): CLOSED and COLLAPSED are terminal — later
// resolutions never reopen or mutate the portal.
func TestPortal_TerminalStateCannotReopen(t *testing.T) {
	t.Run("closed portal stays closed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(100).Decay(0.1).TTL(60 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)

		clk.Advance(60 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusClosed, p.Status)
		snapshot := p

		clk.Advance(120 * time.Second)
		changed, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.False(t, changed)
		require.Equal(t, snapshot, p, "terminal portal must remain entirely unchanged")
	})

	t.Run("collapsed portal stays collapsed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().
			Stable().Energy(10).Decay(1.0).TTL(60 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)

		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)
		snapshot := p

		clk.Advance(600 * time.Second)
		changed, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.False(t, changed)
		require.Equal(t, snapshot, p, "terminal portal must remain entirely unchanged")
	})
}
