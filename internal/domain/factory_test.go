package domain_test

// Stage 2, §20 (04_STAGE_02_PORTAL_CORE_TDD.md): the natural portal
// factory. It only creates a valid NATURAL portal; choosing plane/slot,
// lab mutation, persistence, events and broadcasting are later stages.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// STABILITY-001/003, ENERGY-001/002, PORTAL-002/003 (Final Spec §4, §7,
// §9, §10, §12): deterministic unstable scenario.
// FakeRandom draw order: TTL int → energy float → decay float →
// stability roll float → [hidden offset int] → creatures int.
func TestNewNaturalPortal_GeneratesUnstableWithinSpec(t *testing.T) {
	rnd := testutil.NewFakeRandom().
		QueueInt(10).     // TTL 10s (min allowed)
		QueueFloat(42.5). // initial energy
		QueueFloat(0.7).  // decay
		QueueFloat(0.1).  // stability roll: 0.1 < 0.35 → UNSTABLE
		QueueInt(0).      // hidden offset: opened+5s exactly
		QueueInt(4)       // creatures (max for TTL 10 is 4)

	p := domain.NewNaturalPortal(1, 7, 3, testutil.BaseTime, config.Default(), rnd)

	require.Equal(t, int64(1), p.ID)
	require.Equal(t, "Omenpath #0001", p.Name)
	require.Equal(t, 3, p.SlotIndex)
	require.Equal(t, int64(7), p.DestinationPlaneID)
	require.Equal(t, domain.PortalKindNatural, p.Kind)
	require.Equal(t, domain.PortalStatusOpen, p.Status)
	require.Equal(t, domain.TerminationNone, p.TerminationReason)
	require.Equal(t, domain.PortalFlowNone, p.ObserverFlow)

	require.Equal(t, testutil.BaseTime, p.OpenedAt)
	require.Equal(t, testutil.BaseTime, p.EnergyBaseAt)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), p.ScheduledCloseAt)
	require.InDelta(t, 42.5, p.EnergyBase, 1e-12)
	require.InDelta(t, 0.7, p.EnergyDecayRate, 1e-12)

	require.Equal(t, domain.PortalUnstable, p.Stability)
	require.NotNil(t, p.InstabilityCollapseAt)
	require.Equal(t, testutil.BaseTime.Add(5*time.Second), *p.InstabilityCollapseAt)

	require.Equal(t, 4, p.CreaturesInitial)
	require.Nil(t, p.ClosedAt)
}

// STABILITY-001/002: stable roll produces no hidden timestamp.
func TestNewNaturalPortal_StableHasNoHiddenTimestamp(t *testing.T) {
	rnd := testutil.NewFakeRandom().
		QueueInt(60).     // TTL 60s
		QueueFloat(90).   // energy
		QueueFloat(0.2).  // decay
		QueueFloat(0.99). // roll: ≥ 0.35 → STABLE
		QueueInt(3)       // creatures (max for TTL 60 is 10)

	p := domain.NewNaturalPortal(2, 1, 1, testutil.BaseTime, config.Default(), rnd)

	require.Equal(t, domain.PortalStable, p.Stability)
	require.Nil(t, p.InstabilityCollapseAt)
	require.Equal(t, 3, p.CreaturesInitial)
	require.Equal(t, testutil.BaseTime.Add(60*time.Second), p.ScheduledCloseAt)
}

// Stage 2 plan §13: the hidden window is [opened+5s, scheduled−1s].
func TestNewNaturalPortal_HiddenCollapseWithinWindow(t *testing.T) {
	// TTL 10s → window [5s, 9s]; maximal offset lands exactly at close−1s.
	rnd := testutil.NewFakeRandom().
		QueueInt(10).
		QueueFloat(50).
		QueueFloat(0.5).
		QueueFloat(0.2).                // unstable
		QueueInt(int(4 * time.Second)). // full window span
		QueueInt(0)

	p := domain.NewNaturalPortal(3, 5, 2, testutil.BaseTime, config.Default(), rnd)

	require.Equal(t, domain.PortalUnstable, p.Stability)
	require.NotNil(t, p.InstabilityCollapseAt)
	require.Equal(t, testutil.BaseTime.Add(9*time.Second), *p.InstabilityCollapseAt,
		"hidden collapse must stay at least 1s before natural close")
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), p.ScheduledCloseAt)
}

// CREATURE-001: initial count never exceeds the TTL-derived maximum.
func TestNewNaturalPortal_CreaturesRespectMaxForTTL(t *testing.T) {
	rnd := testutil.NewFakeRandom().
		QueueInt(20). // TTL 20s → max creatures 9
		QueueFloat(70).
		QueueFloat(0.3).
		QueueFloat(0.9). // stable
		QueueInt(9)      // the maximum itself

	p := domain.NewNaturalPortal(4, 2, 5, testutil.BaseTime, config.Default(), rnd)

	require.Equal(t, 9, p.CreaturesInitial)
	require.LessOrEqual(t, p.CreaturesInitial, domain.MaxCreaturesForTTL(20*time.Second, config.Default()))
}
