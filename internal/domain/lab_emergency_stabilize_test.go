package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestStabilizePortalWithLabEnergy_ActiveOverrideCostsZero(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)
	now := testutil.BaseTime.Add(5 * time.Second)
	energyBefore := lab.CurrentEnergy(now, cfg)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg))
	require.Equal(t, energyBefore, lab.CurrentEnergy(now, cfg))
}

func TestStabilizePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg))
	require.Equal(t, domain.PortalStable, portal.Stability)
}

func TestStabilizePortalWithLabEnergy_ActiveOverrideDoesNotRebaseLabEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime.Add(5*time.Second), cfg))
	require.Equal(t, snapshot, lab)
}

func TestStabilizePortalWithLabEnergy_ActiveOverridePreservesDeadline(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	deadline := lab.LeylineOverrideUntil
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg))
	require.Same(t, deadline, lab.LeylineOverrideUntil)
}

func TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsStablePortal(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
}

func TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsOvercharge(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(86)

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrPortalOverchargeRisk)
}

func TestStabilizePortalWithLabEnergy_ActiveOverrideStillRejectsTerminalPortal(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)
	portal.Status = domain.PortalStatusCollapsed

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalBoost(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg))
	require.InDelta(t, 65.0, portal.EnergyBase, 1e-9)
}

func TestStabilizePortalWithLabEnergy_ActiveOverridePreservesPortalDecay(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)
	decay := portal.EnergyDecayRate

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, cfg))
	require.InDelta(t, decay, portal.EnergyDecayRate, 1e-9)
}

func TestStabilizePortalWithLabEnergy_JustBeforeDeadlineIsFree(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab
	portal := unstablePortal(50)
	now := testutil.BaseTime.Add(cfg.EmergencyDuration - time.Nanosecond)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg))
	require.Equal(t, snapshot, lab)
}

func TestStabilizePortalWithLabEnergy_AtDeadlineChargesNormalCost(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg))
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestStabilizePortalWithLabEnergy_ExpiredOverrideInsufficientIsAtomic(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = 0
	lab := overrideLab(0, testutil.BaseTime, cfg)
	labBefore := lab
	portal := unstablePortal(50)
	portalBefore := portal
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg)

	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}
