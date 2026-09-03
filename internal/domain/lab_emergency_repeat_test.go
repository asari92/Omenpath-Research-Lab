package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabState_SecondCollapseResetsRegeneratedEnergyToZero(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	second := testutil.BaseTime.Add(7 * time.Second)
	require.Equal(t, 7, lab.CurrentEnergy(second, cfg))

	require.NoError(t, lab.ActivateLeylineOverride(second, cfg))
	require.Zero(t, lab.CurrentEnergy(second, cfg))
}

func TestLabState_SecondCollapseRebasesAtSecondCollapse(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	second := testutil.BaseTime.Add(7 * time.Second)

	require.NoError(t, lab.ActivateLeylineOverride(second, cfg))
	require.Equal(t, second, lab.EnergyBaseAt)
}

func TestLabState_SecondCollapseReplacesDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	second := testutil.BaseTime.Add(7 * time.Second)

	require.NoError(t, lab.ActivateLeylineOverride(second, cfg))
	require.Equal(t, second.Add(cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestLabState_SecondCollapseDoesNotExtendFromOldDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	oldDeadline := *lab.LeylineOverrideUntil
	second := testutil.BaseTime.Add(7 * time.Second)

	require.NoError(t, lab.ActivateLeylineOverride(second, cfg))
	require.NotEqual(t, oldDeadline.Add(cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestResolvePortalLifecycleWithLabEmergency_SecondCollapseMayUseDifferentCause(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	energyPortal := energyCollapsePortal(5 * time.Second)
	instabilityPortal := instabilityCollapsePortal(12 * time.Second)

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &energyPortal, testutil.BaseTime.Add(5*time.Second), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.TerminationEnergyDepleted, energyPortal.TerminationReason)

	changed, err = domain.ResolvePortalLifecycleWithLabEmergency(&lab, &instabilityPortal, testutil.BaseTime.Add(12*time.Second), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.TerminationInstability, instabilityPortal.TerminationReason)
	require.Equal(t, testutil.BaseTime.Add(12*time.Second), lab.EnergyBaseAt)
}

func TestLabState_SecondCollapseAtSameTimestampIsDeterministic(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	first := lab

	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	require.Equal(t, first, lab)
}

func TestLabState_OutOfOrderCollapseRejectsWithoutMutation(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime.Add(10*time.Second), cfg))
	snapshot := lab

	err := lab.ActivateLeylineOverride(testutil.BaseTime.Add(9*time.Second), cfg)

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, lab)
}

func TestResolvePortalLifecycleWithLabEmergency_TwoPortalsResetTwice(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	first := energyCollapsePortal(5 * time.Second)
	second := energyCollapsePortal(12 * time.Second)

	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &first, testutil.BaseTime.Add(5*time.Second), cfg)
	require.NoError(t, err)
	require.Equal(t, 7, lab.CurrentEnergy(testutil.BaseTime.Add(12*time.Second), cfg))

	_, err = domain.ResolvePortalLifecycleWithLabEmergency(&lab, &second, testutil.BaseTime.Add(12*time.Second), cfg)
	require.NoError(t, err)
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, testutil.BaseTime.Add(12*time.Second), lab.EnergyBaseAt)
}

func TestResolvePortalLifecycleWithLabEmergency_ReplayOfSecondPortalIsIdempotent(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := energyCollapsePortal(12 * time.Second)
	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(12*time.Second), cfg)
	require.NoError(t, err)
	labBefore := lab
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(15*time.Second), cfg)

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}
