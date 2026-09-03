package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
)

func TestPortalBuilder_Defaults(t *testing.T) {
	p := NewPortalBuilder().Build()

	require.Equal(t, int64(1), p.ID)
	require.Equal(t, "Omenpath #0001", p.Name)
	require.Equal(t, 1, p.SlotIndex)
	require.Equal(t, domain.PortalKindNatural, p.Kind)
	require.Equal(t, domain.PortalStatusOpen, p.Status)
	require.Equal(t, domain.TerminationNone, p.TerminationReason)
	require.Equal(t, domain.PortalStable, p.Stability)
	require.Nil(t, p.InstabilityCollapseAt)
	require.Equal(t, domain.PortalFlowNone, p.ObserverFlow)
	require.Equal(t, BaseTime, p.OpenedAt)
	require.Equal(t, BaseTime, p.EnergyBaseAt)
	require.Equal(t, BaseTime.Add(60*time.Second), p.ScheduledCloseAt)
	require.Equal(t, 100.0, p.EnergyBase)
	require.Equal(t, 0.5, p.EnergyDecayRate)
	require.Equal(t, 0, p.CreaturesInitial)
	require.Nil(t, p.ClosedAt)
}

func TestPortalBuilder_AppliesOverrides(t *testing.T) {
	p := NewPortalBuilder().
		Stable().
		Energy(80).
		Decay(0.25).
		TTL(45 * time.Second).
		Creatures(3).
		Build()

	require.Equal(t, 80.0, p.EnergyBase)
	require.Equal(t, 0.25, p.EnergyDecayRate)
	require.Equal(t, BaseTime.Add(45*time.Second), p.ScheduledCloseAt)
	require.Equal(t, 3, p.CreaturesInitial)
	// EnergyBaseAt / OpenedAt stay anchored to the deterministic base time.
	require.Equal(t, BaseTime, p.EnergyBaseAt)
	require.Equal(t, BaseTime, p.OpenedAt)
}

func TestPortalBuilder_UnstableSetsExplicitHiddenCollapse(t *testing.T) {
	hidden := BaseTime.Add(23 * time.Second)
	p := NewPortalBuilder().Unstable(hidden).Build()

	require.Equal(t, domain.PortalUnstable, p.Stability)
	require.NotNil(t, p.InstabilityCollapseAt)
	require.Equal(t, hidden, *p.InstabilityCollapseAt)
}
