package domain_test

// Stage 2, substages 2.2–2.4 (04_STAGE_02_PORTAL_CORE_TDD.md):
// portal energy derivation, depletion timestamp, natural/energy ordering.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// ENERGY-003 / ENERGY-004 (Final Spec §9): current energy is derived from
// baseline + elapsed time and clamped at zero.
func TestPortal_CurrentEnergy(t *testing.T) {
	cases := []struct {
		name     string
		base     float64
		decay    float64
		elapsed  time.Duration
		expected float64
	}{
		{"no elapsed time", 80, 0.5, 0, 80},
		{"linear decay", 80, 0.5, 10 * time.Second, 75},
		{"reaches zero exactly", 10, 1.0, 10 * time.Second, 0},
		{"clamped at zero", 10, 1.0, 15 * time.Second, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testutil.NewPortalBuilder().Stable().Energy(tc.base).Decay(tc.decay).Build()
			now := testutil.BaseTime.Add(tc.elapsed)
			require.InDelta(t, tc.expected, p.CurrentEnergy(now), 1e-9)
		})
	}
}

// Stage 2 plan §10: energy depletion moment is a pure calculation.
func TestPortal_EnergyDepletionAt(t *testing.T) {
	t.Run("base 80 decay 0.5 depletes at +160s", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(80).Decay(0.5).Build()
		at, ok := p.EnergyDepletionAt()
		require.True(t, ok)
		require.Equal(t, testutil.BaseTime.Add(160*time.Second), at)
	})

	t.Run("base 10 decay 1.0 depletes at +10s", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(10).Decay(1.0).Build()
		at, ok := p.EnergyDepletionAt()
		require.True(t, ok)
		require.Equal(t, testutil.BaseTime.Add(10*time.Second), at)
	})

	t.Run("zero decay never depletes (defensive)", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(80).Decay(0).Build()
		_, ok := p.EnergyDepletionAt()
		require.False(t, ok)
	})
}

// ScheduledRemaining is the derived TTL countdown, clamped at zero.
func TestPortal_ScheduledRemaining(t *testing.T) {
	p := testutil.NewPortalBuilder().TTL(60 * time.Second).Build()

	require.Equal(t, 60*time.Second, p.ScheduledRemaining(testutil.BaseTime))
	require.Equal(t, 10*time.Second, p.ScheduledRemaining(testutil.BaseTime.Add(50*time.Second)))
	require.Equal(t, time.Duration(0), p.ScheduledRemaining(testutil.BaseTime.Add(60*time.Second)))
	require.Equal(t, time.Duration(0), p.ScheduledRemaining(testutil.BaseTime.Add(70*time.Second)))
}

// PORTAL-005 (Final Spec §8): the semantic event time is preserved even
// when a delayed resolution notices the expiry late (Stage 2 plan §9).
func TestPortal_LateResolutionPreservesNaturalCloseTime(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().Energy(100).Decay(0.1). // depletion at +1000s
		TTL(60 * time.Second).
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(100 * time.Second) // 40s after the scheduled close

	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
	require.NotNil(t, p.ClosedAt)
	require.Equal(t, testutil.BaseTime.Add(60*time.Second), *p.ClosedAt,
		"ClosedAt must be the scheduled moment, not the lazy resolution time")
}

// Stage 2 plan Rule B: energy hitting zero exactly at natural close time
// is a NATURAL_CLOSE, not a collapse.
func TestPortal_NaturalCloseWinsEnergyTie(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().Energy(10).Decay(1.0). // depletion exactly at +10s
		TTL(10 * time.Second).          // close exactly at +10s
		Build()

	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(10 * time.Second)

	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationNaturalClose, p.TerminationReason)
}

// Stage 2 plan §11: natural close scheduled earlier than energy depletion.
func TestPortal_NaturalCloseBeforeEnergyDepletion(t *testing.T) {
	p := testutil.NewPortalBuilder().
		Stable().Energy(100).Decay(0.1). // depletion at +1000s
		TTL(10 * time.Second).           // close at +10s
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
