package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabState_LabEnergyRegeneratesDuringOverride(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)

	require.True(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(5*time.Second), cfg))
	require.Equal(t, 5, lab.CurrentEnergy(testutil.BaseTime.Add(5*time.Second), cfg))
}

func TestLabState_OverrideRegenerationUsesCompletedWholeSeconds(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)

	require.Equal(t, 0, lab.CurrentEnergy(testutil.BaseTime.Add(999*time.Millisecond), cfg))
	require.Equal(t, 1, lab.CurrentEnergy(testutil.BaseTime.Add(time.Second), cfg))
}

func TestLabState_OverrideRegenerationCapsAtMaximum(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.LabEnergyMax
	lab := overrideLab(0, testutil.BaseTime, cfg)
	now := testutil.BaseTime.Add(time.Second)

	require.True(t, lab.LeylineOverrideActive(now, cfg))
	require.Equal(t, cfg.LabEnergyMax, lab.CurrentEnergy(now, cfg))
}

func TestLabState_FreeCloseDoesNotInterruptRegeneration(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime.Add(5*time.Second), false, cfg))
	require.Equal(t, 6, lab.CurrentEnergy(testutil.BaseTime.Add(6*time.Second), cfg))
}

func TestLabState_FreeStabilizeDoesNotInterruptRegeneration(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime.Add(5*time.Second), cfg))
	require.Equal(t, 6, lab.CurrentEnergy(testutil.BaseTime.Add(6*time.Second), cfg))
}

func TestLabState_ExtractionCostDuringOverrideIsThirty(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)

	require.True(t, lab.LeylineOverrideActive(testutil.BaseTime, cfg))
	require.Equal(t, 30, cfg.ExtractionCost)
}

func TestLabState_ExtractionSpendDuringOverrideAllowsExactThirty(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost
	lab := overrideLab(0, testutil.BaseTime, cfg)
	now := testutil.BaseTime.Add(time.Second)

	require.NoError(t, lab.SpendEnergy(now, cfg.ExtractionCost, cfg))
	require.Zero(t, lab.EnergyBase)
}

func TestLabState_ExtractionSpendDuringOverrideRejectsTwentyNine(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost - 1
	lab := overrideLab(0, testutil.BaseTime, cfg)
	now := testutil.BaseTime.Add(time.Second)

	err := lab.SpendEnergy(now, cfg.ExtractionCost, cfg)
	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
}

func TestLabState_ExtractionSpendDuringOverrideDoesNotChangeDeadline(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost
	lab := overrideLab(0, testutil.BaseTime, cfg)
	deadline := lab.LeylineOverrideUntil

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime.Add(time.Second), cfg.ExtractionCost, cfg))
	require.Same(t, deadline, lab.LeylineOverrideUntil)
}

func TestLabState_ExtractionRebaseDoesNotChangeActiveWindow(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost
	lab := overrideLab(0, testutil.BaseTime, cfg)

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime.Add(time.Second), cfg.ExtractionCost, cfg))
	require.True(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(2*time.Second), cfg))
	require.False(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(cfg.EmergencyDuration), cfg))
}

func TestLabState_ExtractionCostCharacterizationDoesNotCreatePortal(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost
	lab := overrideLab(0, testutil.BaseTime, cfg)
	var portals []domain.Portal

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime.Add(time.Second), cfg.ExtractionCost, cfg))
	require.Empty(t, portals)
}
