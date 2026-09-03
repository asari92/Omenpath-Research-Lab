package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func labAt(energy int, at time.Time) domain.LabState {
	return domain.LabState{EnergyBase: energy, EnergyBaseAt: at}
}

func TestNewLabState_AcceptsZero(t *testing.T) {
	lab, err := domain.NewLabState(0, testutil.BaseTime, config.Default())

	require.NoError(t, err)
	require.Equal(t, 0, lab.EnergyBase)
}

func TestNewLabState_AcceptsMaximum(t *testing.T) {
	cfg := config.Default()
	lab, err := domain.NewLabState(cfg.LabEnergyMax, testutil.BaseTime, cfg)

	require.NoError(t, err)
	require.Equal(t, cfg.LabEnergyMax, lab.EnergyBase)
}

func TestNewLabState_RejectsBelowZero(t *testing.T) {
	_, err := domain.NewLabState(-1, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
}

func TestNewLabState_RejectsAboveMaximum(t *testing.T) {
	cfg := config.Default()
	_, err := domain.NewLabState(cfg.LabEnergyMax+1, testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
}

func TestNewTutorialLabState_StartsAtMaximum(t *testing.T) {
	cfg := config.Default()
	lab := domain.NewTutorialLabState(testutil.BaseTime, cfg)
	require.Equal(t, cfg.LabEnergyMax, lab.EnergyBase)
}

func TestNewTutorialLabState_UsesProvidedTimestamp(t *testing.T) {
	lab := domain.NewTutorialLabState(testutil.BaseTime, config.Default())
	require.Equal(t, testutil.BaseTime, lab.EnergyBaseAt)
}

func TestNewTutorialLabState_HasNoOverride(t *testing.T) {
	lab := domain.NewTutorialLabState(testutil.BaseTime, config.Default())
	require.Nil(t, lab.LeylineOverrideUntil)
}

func TestLabState_CurrentEnergyAtBaseline(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	require.Equal(t, 40, lab.CurrentEnergy(testutil.BaseTime, config.Default()))
}

func TestLabState_CurrentEnergyUsesCompletedWholeSeconds(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	require.Equal(t, 40, lab.CurrentEnergy(testutil.BaseTime.Add(999*time.Millisecond), config.Default()))
}

func TestLabState_CurrentEnergyAtExactSecond(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	require.Equal(t, 41, lab.CurrentEnergy(testutil.BaseTime.Add(time.Second), config.Default()))
}

func TestLabState_CurrentEnergyUsesConfiguredRate(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = 3
	lab := labAt(40, testutil.BaseTime)
	require.Equal(t, 49, lab.CurrentEnergy(testutil.BaseTime.Add(3*time.Second), cfg))
}

func TestLabState_CurrentEnergyCapsAtMaximum(t *testing.T) {
	cfg := config.Default()
	lab := labAt(99, testutil.BaseTime)
	require.Equal(t, cfg.LabEnergyMax, lab.CurrentEnergy(testutil.BaseTime.Add(time.Minute), cfg))
}

func TestLabState_CurrentEnergyBeforeBaselineDoesNotRegenerate(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	require.Equal(t, 40, lab.CurrentEnergy(testutil.BaseTime.Add(-time.Second), config.Default()))
}

func TestLabState_CurrentEnergyDoesNotMutateBaseline(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab

	_ = lab.CurrentEnergy(testutil.BaseTime.Add(10*time.Second), config.Default())

	require.Equal(t, snapshot, lab)
}

func TestLabState_CurrentEnergyIsInteger(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	energy := lab.CurrentEnergy(testutil.BaseTime.Add(10*time.Second+900*time.Millisecond), config.Default())
	require.IsType(t, int(0), energy)
	require.Equal(t, 50, energy)
}
