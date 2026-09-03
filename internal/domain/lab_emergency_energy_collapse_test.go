package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func energyCollapsePortal(collapseAfter time.Duration) domain.Portal {
	return testutil.NewPortalBuilder().
		Stable().
		Energy(collapseAfter.Seconds()).
		Decay(1).
		TTL(time.Minute).
		Build()
}

func TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionResetsLab(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(10 * time.Second)

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusCollapsed, portal.Status)
	require.Equal(t, domain.TerminationEnergyDepleted, portal.TerminationReason)
	require.Zero(t, lab.EnergyBase)
}

func TestResolvePortalLifecycleWithLabEmergency_EnergyDepletionStartsOverride(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(10 * time.Second)
	collapseAt := testutil.BaseTime.Add(10 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, collapseAt, cfg)

	require.NoError(t, err)
	require.True(t, lab.LeylineOverrideActive(collapseAt, cfg))
}

func TestResolvePortalLifecycleWithLabEmergency_UsesPortalClosedAt(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(10 * time.Second)
	resolvedAt := testutil.BaseTime.Add(15 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, resolvedAt, cfg)

	require.NoError(t, err)
	require.NotNil(t, portal.ClosedAt)
	require.Equal(t, *portal.ClosedAt, lab.EnergyBaseAt)
	require.Equal(t, portal.ClosedAt.Add(cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestResolvePortalLifecycleWithLabEmergency_LateResolutionIncludesRegeneration(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(10 * time.Second)
	resolvedAt := testutil.BaseTime.Add(15 * time.Second)

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, resolvedAt, cfg)

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, 5, lab.CurrentEnergy(resolvedAt, cfg))
}

func TestResolvePortalLifecycleWithLabEmergency_BeforeDeadlineChangesNothing(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(10 * time.Second)
	labBefore := lab
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second-time.Nanosecond), config.Default())

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}

func TestResolvePortalLifecycleWithLabEmergency_NaturalCloseDoesNotStartOverride(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	labBefore := lab
	portal := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(10 * time.Second).Build()

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), config.Default())

	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.TerminationNaturalClose, portal.TerminationReason)
	require.Equal(t, labBefore, lab)
}

func TestResolvePortalLifecycleWithLabEmergency_AlreadyCollapsedDoesNotResetAgain(t *testing.T) {
	portal := energyCollapsePortal(10 * time.Second)
	_, err := portal.ResolveLifecycle(testutil.BaseTime.Add(10 * time.Second))
	require.NoError(t, err)
	lab := labAt(40, testutil.BaseTime.Add(15*time.Second))
	labBefore := lab

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(20*time.Second), config.Default())

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
}

func TestResolvePortalLifecycleWithLabEmergency_AlreadyClosedDoesNotChangeLab(t *testing.T) {
	portal := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(10 * time.Second).Build()
	_, err := portal.ResolveLifecycle(testutil.BaseTime.Add(10 * time.Second))
	require.NoError(t, err)
	lab := labAt(40, testutil.BaseTime.Add(15*time.Second))
	labBefore := lab

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(20*time.Second), config.Default())

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
}

func TestResolvePortalLifecycleWithLabEmergency_InvalidLabIsAtomic(t *testing.T) {
	lab := labAt(40, testutil.BaseTime.Add(20*time.Second))
	portal := energyCollapsePortal(10 * time.Second)
	labBefore := lab
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(20*time.Second), config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}

func TestResolvePortalLifecycleWithLabEmergency_NilPortalDoesNotMutateLab(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	labBefore := lab

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, nil, testutil.BaseTime, config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
}
