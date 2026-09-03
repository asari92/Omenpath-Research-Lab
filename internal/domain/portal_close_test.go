package domain_test

// Stage 2, substage 2.8 (04_STAGE_02_PORTAL_CORE_TDD.md): manual close
// primitive. Final Spec §21: cost and observer-in-transit confirmation are
// orchestrated later; here only state transition + creature confirmation.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// PORTAL-006 (Final Spec §21): normal manual close without creatures.
func TestPortal_ManualCloseWithoutCreatures(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).TTL(60 * time.Second).Build()

	now := testutil.BaseTime.Add(5 * time.Second)
	require.NoError(t, p.Close(now, false, config.Default()))

	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationManualClose, p.TerminationReason)
	require.NotNil(t, p.ClosedAt)
	require.Equal(t, now, *p.ClosedAt) // manual action time is the semantic time
	require.Equal(t, now, p.UpdatedAt)
}

// CREATURE-008 (Final Spec §12/§21): creatures inside demand confirmation;
// without it the portal remains entirely unchanged.
func TestPortal_ManualCloseRequiresConfirmationWithCreatures(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).
		TTL(60 * time.Second).Creatures(2).Build()
	snapshot := p

	now := testutil.BaseTime // creatures still inside (2 → 2)
	err := p.Close(now, false, config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, snapshot, p)
	require.Equal(t, domain.PortalStatusOpen, p.Status)
}

func TestPortal_ManualCloseWithConfirmationClosesDespiteCreatures(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).
		TTL(60 * time.Second).Creatures(2).Build()

	now := testutil.BaseTime
	require.NoError(t, p.Close(now, true, config.Default()))

	require.Equal(t, domain.PortalStatusClosed, p.Status)
	require.Equal(t, domain.TerminationManualClose, p.TerminationReason)
}

// Creatures derived from the initial count: once they have all passed, no
// confirmation is needed anymore.
func TestPortal_ManualCloseAfterCreaturesPassedNeedsNoConfirmation(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).
		TTL(120 * time.Second).Creatures(2).Build()

	now := testutil.BaseTime.Add(10 * time.Second) // both creatures passed
	require.Equal(t, 0, p.CreaturesInside(now, config.Default()))
	require.NoError(t, p.Close(now, false, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, p.Status)
}

// Stage 2 plan §16: terminal portals reject Close.
func TestPortal_ManualCloseRejectsTerminalPortal(t *testing.T) {
	t.Run("closed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).TTL(10 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		snapshot := p

		require.ErrorIs(t, p.Close(clk.Now(), false, config.Default()), domain.ErrPortalNotOpen)
		require.Equal(t, snapshot, p)
	})

	t.Run("collapsed", func(t *testing.T) {
		p := testutil.NewPortalBuilder().Stable().Energy(10).Decay(1.0).TTL(60 * time.Second).Build()
		clk := testutil.NewFakeClock(testutil.BaseTime)
		clk.Advance(10 * time.Second)
		_, err := p.ResolveLifecycle(clk.Now())
		require.NoError(t, err)
		require.Equal(t, domain.PortalStatusCollapsed, p.Status)
		snapshot := p

		require.ErrorIs(t, p.Close(clk.Now(), false, config.Default()), domain.ErrPortalNotOpen)
		require.Equal(t, snapshot, p)
	})
}

// A portal closed manually must never be reopened by later lifecycle
// resolution (PORTAL-009 through the manual path).
func TestPortal_ManuallyClosedPortalIsTerminalForLifecycle(t *testing.T) {
	p := testutil.NewPortalBuilder().Stable().Energy(50).Decay(0.5).TTL(60 * time.Second).Build()
	clk := testutil.NewFakeClock(testutil.BaseTime)

	require.NoError(t, p.Close(clk.Now(), false, config.Default()))
	snapshot := p

	clk.Advance(120 * time.Second) // past scheduled close and depletion
	changed, err := p.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, snapshot, p)
}
