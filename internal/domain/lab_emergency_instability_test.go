package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func instabilityCollapsePortal(collapseAfter time.Duration) domain.Portal {
	return testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(collapseAfter)).
		Energy(100).
		Decay(0.1).
		TTL(time.Minute).
		Build()
}

func TestResolvePortalLifecycleWithLabEmergency_InstabilityResetsLab(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	portal := instabilityCollapsePortal(10 * time.Second)

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, portal.Status)
	require.Equal(t, domain.TerminationInstability, portal.TerminationReason)
	require.Zero(t, lab.EnergyBase)
}

func TestResolvePortalLifecycleWithLabEmergency_InstabilityStartsOverride(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := instabilityCollapsePortal(10 * time.Second)
	collapseAt := testutil.BaseTime.Add(10 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, collapseAt, cfg)

	require.NoError(t, err)
	require.True(t, lab.LeylineOverrideActive(collapseAt, cfg))
}

func TestResolvePortalLifecycleWithLabEmergency_InstabilityUsesHiddenCollapseTime(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := instabilityCollapsePortal(10 * time.Second)
	resolvedAt := testutil.BaseTime.Add(15 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, resolvedAt, cfg)

	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), lab.EnergyBaseAt)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second+cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestResolvePortalLifecycleWithLabEmergency_LateInstabilityResolutionRegenerates(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := instabilityCollapsePortal(10 * time.Second)
	resolvedAt := testutil.BaseTime.Add(15 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, resolvedAt, cfg)

	require.NoError(t, err)
	require.Equal(t, 5, lab.CurrentEnergy(resolvedAt, cfg))
}

func TestResolvePortalLifecycleWithLabEmergency_NaturalCloseTieDoesNotStartOverride(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	labBefore := lab
	portal := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(10 * time.Second)).
		Energy(100).Decay(0.1).TTL(10 * time.Second).Build()

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.TerminationNaturalClose, portal.TerminationReason)
	require.Equal(t, labBefore, lab)
}

func TestResolvePortalLifecycleWithLabEmergency_EnergyInstabilityTieStartsOneOverride(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(10 * time.Second)).
		Energy(10).Decay(1).TTL(time.Minute).Build()

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), cfg)

	require.NoError(t, err)
	require.True(t, changed)
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second+cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestResolvePortalLifecycleWithLabEmergency_PreservesEnergyDepletedTieReason(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	portal := testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(10 * time.Second)).
		Energy(10).Decay(1).TTL(time.Minute).Build()

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.TerminationEnergyDepleted, portal.TerminationReason)
}

func TestResolvePortalLifecycleWithLabEmergency_StablePortalCannotCollapseByHiddenTime(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	labBefore := lab
	portal := instabilityCollapsePortal(10 * time.Second)
	portal.Stability = domain.PortalStable
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, labBefore, lab)
}
