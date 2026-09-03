package domain_test

// Stage 2, substage 2.9 + §17–18 (04_STAGE_02_PORTAL_CORE_TDD.md):
// risk lifetimes, formula boundaries, instability penalty and the
// "terminal portals have no current risk" rule (Final Spec §13).

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func cfg() config.Config { return config.Default() }

// RISK-001 (Final Spec §13): energy lifetime = current energy / decay.
func TestPortal_EnergyLifetime(t *testing.T) {
	t.Run("energy 50 decay 0.5 → 100s", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(50).Decay(0.5).Build()
		require.Equal(t, 100*time.Second, p.EnergyLifetime(testutil.BaseTime))
	})

	t.Run("energy 10 decay 1.0 → 10s", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(10).Decay(1.0).Build()
		require.Equal(t, 10*time.Second, p.EnergyLifetime(testutil.BaseTime))
	})
}

// RISK-002 (Final Spec §13): effective lifetime = min(scheduled, energy).
func TestPortal_EffectiveLifetime(t *testing.T) {
	t.Run("scheduled 80 limits energy lifetime 100", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(50).Decay(0.5).TTL(80 * time.Second).Build()
		require.Equal(t, 80*time.Second, p.EffectiveLifetime(testutil.BaseTime))
	})

	t.Run("energy lifetime 10 limits scheduled 120", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Energy(10).Decay(1.0).TTL(120 * time.Second).Build()
		require.Equal(t, 10*time.Second, p.EffectiveLifetime(testutil.BaseTime))
	})
}

// RISK-003/005 (Final Spec §13): base formula with the 45s horizon and the
// 100 cap. Stable portal, energy lifetime kept far away.
func TestPortal_RiskScore(t *testing.T) {
	cases := []struct {
		name      string
		remaining time.Duration
		unstable  bool
		expected  float64
	}{
		{"safe horizon and beyond → 0", 60 * time.Second, false, 0},
		{"exactly safe horizon → 0", 45 * time.Second, false, 0},
		{"boundary 33.75s → 25", 33750 * time.Millisecond, false, 25},
		{"boundary 22.5s → 50", 22500 * time.Millisecond, false, 50},
		{"boundary 11.25s → 75", 11250 * time.Millisecond, false, 75},
		{"34s → 24.44 LOW territory", 34 * time.Second, false, (45.0 - 34.0) / 45.0 * 100},
		{"11s → 75.55", 11 * time.Second, false, (45.0 - 11.0) / 45.0 * 100},
		{"effective 0 → 100", 0, false, 100},
		{"unstable at horizon → 20 penalty", 60 * time.Second, true, 20},
		{"unstable adds 20", 30 * time.Second, true, (45.0-30.0)/45.0*100 + 20},
		{"unstable at 0 capped at 100", 0, true, 100},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := testutil.NewPortalBuilder().Energy(100).Decay(0.1).TTL(tc.remaining)
			if tc.unstable {
				b = b.Unstable(testutil.BaseTime.Add(tc.remaining - time.Second))
			} else {
				b = b.Stable()
			}
			p := b.Build()
			require.InDelta(t, tc.expected, p.RiskScore(testutil.BaseTime, cfg()), 1e-9)
		})
	}
}

// RISK-006..009 (Final Spec §13): exact band boundaries.
func TestPortal_RiskLevelBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		remaining time.Duration
		expected  domain.RiskLevel
	}{
		{"score 0 → LOW", 45 * time.Second, domain.RiskLow},
		{"score 24.4 → LOW", 34 * time.Second, domain.RiskLow},
		{"score exactly 25 → LOW", 33750 * time.Millisecond, domain.RiskLow},
		{"score just above 25 → MEDIUM", 33 * time.Second, domain.RiskMedium},
		{"score exactly 50 → MEDIUM", 22500 * time.Millisecond, domain.RiskMedium},
		{"score just above 50 → HIGH", 22 * time.Second, domain.RiskHigh},
		{"score exactly 75 → HIGH", 11250 * time.Millisecond, domain.RiskHigh},
		{"score just above 75 → CRITICAL", 11 * time.Second, domain.RiskCritical},
		{"score 100 → CRITICAL", 0, domain.RiskCritical},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testutil.NewPortalBuilder().Stable().
				Energy(100).Decay(0.1).TTL(tc.remaining).Build()
			level, ok := p.RiskLevel(testutil.BaseTime, cfg())
			require.True(t, ok)
			require.Equal(t, tc.expected, level)
		})
	}
}

// Stage 2 plan §17.4 agreed examples.
func TestPortal_RiskAgreedExamples(t *testing.T) {
	t.Run("effective 14s stable → HIGH ~68.89", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(14 * time.Second).Build()
		require.InDelta(t, (45.0-14.0)/45.0*100, p.RiskScore(testutil.BaseTime, cfg()), 1e-9)
		level, ok := p.RiskLevel(testutil.BaseTime, cfg())
		require.True(t, ok)
		require.Equal(t, domain.RiskHigh, level)
	})

	t.Run("effective 10s stable → CRITICAL ~77.78", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(10 * time.Second).Build()
		require.InDelta(t, (45.0-10.0)/45.0*100, p.RiskScore(testutil.BaseTime, cfg()), 1e-9)
		level, ok := p.RiskLevel(testutil.BaseTime, cfg())
		require.True(t, ok)
		require.Equal(t, domain.RiskCritical, level)
	})
}

// RISK-004 (Final Spec §13): instability penalty shifts the level.
func TestPortal_UnstableAddsRiskPenalty(t *testing.T) {
	stable := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(30 * time.Second).Build()
	unstable := testutil.NewPortalBuilder().Unstable(testutil.BaseTime.Add(29 * time.Second)).
		Energy(100).Decay(0.1).TTL(30 * time.Second).Build()

	require.InDelta(t, (45.0-30.0)/45.0*100, stable.RiskScore(testutil.BaseTime, cfg()), 1e-9)
	require.InDelta(t, (45.0-30.0)/45.0*100+20, unstable.RiskScore(testutil.BaseTime, cfg()), 1e-9)

	stableLevel, ok := stable.RiskLevel(testutil.BaseTime, cfg())
	require.True(t, ok)
	require.Equal(t, domain.RiskMedium, stableLevel)

	unstableLevel, ok := unstable.RiskLevel(testutil.BaseTime, cfg())
	require.True(t, ok)
	require.Equal(t, domain.RiskHigh, unstableLevel)
}

// RISK-010 / STABILITY-004 (Final Spec §13): the hidden instability
// timestamp itself must never influence the risk.
func TestPortal_HiddenInstabilityTimestampDoesNotAffectRisk(t *testing.T) {
	early := testutil.NewPortalBuilder().Unstable(testutil.BaseTime.Add(6 * time.Second)).
		Energy(100).Decay(0.1).TTL(60 * time.Second).Build()
	late := testutil.NewPortalBuilder().Unstable(testutil.BaseTime.Add(59 * time.Second)).
		Energy(100).Decay(0.1).TTL(60 * time.Second).Build()

	require.InDelta(t, early.RiskScore(testutil.BaseTime, cfg()),
		late.RiskScore(testutil.BaseTime, cfg()), 1e-12)

	earlyLevel, ok := early.RiskLevel(testutil.BaseTime, cfg())
	require.True(t, ok)
	lateLevel, ok := late.RiskLevel(testutil.BaseTime, cfg())
	require.True(t, ok)
	require.Equal(t, earlyLevel, lateLevel)
}

// Stage 2 plan §18 (resolved decision S2-D2): risk is defined only for
// OPEN portals — terminal ones report no current level.
func TestPortal_RiskLevelUndefinedForTerminalPortals(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(10 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)

		level, ok := p.RiskLevel(clk.Now(), cfg())
		require.False(t, ok, "terminal portal must not report a current risk level")
		require.Empty(t, level)
	})

	t.Run("collapsed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(10).Decay(1.0).TTL(60 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)

		level, ok := p.RiskLevel(clk.Now(), cfg())
		require.False(t, ok)
		require.Empty(t, level)
	})
}
