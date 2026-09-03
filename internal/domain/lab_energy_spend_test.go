package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabState_SpendEnergySubtractsCost(t *testing.T) {
	lab := labAt(50, testutil.BaseTime)
	require.NoError(t, lab.SpendEnergy(testutil.BaseTime, 20, config.Default()))
	require.Equal(t, 30, lab.EnergyBase)
}

func TestLabState_SpendEnergyUsesRegeneratedCurrentValue(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	now := testutil.BaseTime.Add(3 * time.Second)
	require.NoError(t, lab.SpendEnergy(now, 5, config.Default()))
	require.Equal(t, 8, lab.EnergyBase)
}

func TestLabState_SpendEnergyRebasesAtNow(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	now := testutil.BaseTime.Add(3 * time.Second)
	require.NoError(t, lab.SpendEnergy(now, 5, config.Default()))
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestLabState_SpendEnergyPreventsDoubleCountingRegeneration(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	now := testutil.BaseTime.Add(3 * time.Second)
	require.NoError(t, lab.SpendEnergy(now, 5, config.Default()))
	require.Equal(t, 9, lab.CurrentEnergy(now.Add(time.Second), config.Default()))
}

func TestLabState_SpendEnergyAllowsExactBalance(t *testing.T) {
	lab := labAt(20, testutil.BaseTime)
	require.NoError(t, lab.SpendEnergy(testutil.BaseTime, 20, config.Default()))
	require.Zero(t, lab.EnergyBase)
}

func TestLabState_SpendEnergyRejectsInsufficientBalance(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	err := lab.SpendEnergy(testutil.BaseTime, 20, config.Default())
	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
}

func TestLabState_SpendEnergyInsufficientIsAtomic(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	snapshot := lab

	_ = lab.SpendEnergy(testutil.BaseTime, 20, config.Default())

	require.Equal(t, snapshot, lab)
}

func TestLabState_SpendEnergyZeroDoesNotRebase(t *testing.T) {
	override := testutil.BaseTime.Add(time.Minute)
	lab := labAt(40, testutil.BaseTime)
	lab.LeylineOverrideUntil = &override
	snapshot := lab

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime.Add(10*time.Second), 0, config.Default()))
	require.Equal(t, snapshot, lab)
}

func TestLabState_SpendEnergyRejectsNegativeCost(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab

	err := lab.SpendEnergy(testutil.BaseTime, -1, config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, lab)
}

func TestLabState_SpendEnergyRejectsMalformedBaseline(t *testing.T) {
	lab := labAt(config.Default().LabEnergyMax+1, testutil.BaseTime)
	snapshot := lab

	err := lab.SpendEnergy(testutil.BaseTime, 1, config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, lab)
}

func TestLabState_SpendEnergyRejectsBackwardTime(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab

	err := lab.SpendEnergy(testutil.BaseTime.Add(-time.Second), 1, config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, lab)
}

func TestLabState_CanAffordUsesDerivedEnergy(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	now := testutil.BaseTime.Add(10 * time.Second)

	require.True(t, lab.CanAfford(now, 20, config.Default()))
	require.False(t, lab.CanAfford(now, 21, config.Default()))
}
