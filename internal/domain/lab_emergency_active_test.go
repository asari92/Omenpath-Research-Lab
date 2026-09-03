package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabState_ActivateLeylineOverrideResetsEnergyToZero(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, config.Default()))
	require.Zero(t, lab.EnergyBase)
}

func TestLabState_ActivateLeylineOverrideUsesCollapseTimestampAsBaseline(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	collapseAt := testutil.BaseTime.Add(5 * time.Second)
	require.NoError(t, lab.ActivateLeylineOverride(collapseAt, config.Default()))
	require.Equal(t, collapseAt, lab.EnergyBaseAt)
}

func TestLabState_ActivateLeylineOverrideSetsConfiguredDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	collapseAt := testutil.BaseTime.Add(5 * time.Second)

	require.NoError(t, lab.ActivateLeylineOverride(collapseAt, cfg))
	require.NotNil(t, lab.LeylineOverrideUntil)
	require.Equal(t, collapseAt.Add(cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestLabState_ActivateLeylineOverrideReplacesExistingDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	secondCollapse := testutil.BaseTime.Add(5 * time.Second)

	require.NoError(t, lab.ActivateLeylineOverride(secondCollapse, cfg))
	require.Equal(t, secondCollapse.Add(cfg.EmergencyDuration), *lab.LeylineOverrideUntil)
}

func TestLabState_ActivateLeylineOverrideRejectsBackwardCollapseTime(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	err := lab.ActivateLeylineOverride(testutil.BaseTime.Add(-time.Nanosecond), config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
}

func TestLabState_ActivateLeylineOverrideRejectsInvalidBaseline(t *testing.T) {
	lab := labAt(config.Default().LabEnergyMax+1, testutil.BaseTime)
	err := lab.ActivateLeylineOverride(testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
}

func TestLabState_ActivateLeylineOverrideRejectsNonPositiveDuration(t *testing.T) {
	cfg := config.Default()
	cfg.EmergencyDuration = 0
	lab := labAt(75, testutil.BaseTime)

	err := lab.ActivateLeylineOverride(testutil.BaseTime, cfg)
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
}

func TestLabState_ActivateLeylineOverrideFailureIsAtomic(t *testing.T) {
	deadline := testutil.BaseTime.Add(time.Minute)
	lab := labAt(75, testutil.BaseTime)
	lab.LeylineOverrideUntil = &deadline
	snapshot := lab

	_ = lab.ActivateLeylineOverride(testutil.BaseTime.Add(-time.Nanosecond), config.Default())
	require.Equal(t, snapshot, lab)
}

func TestLabState_LeylineOverrideActiveAtStart(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	require.True(t, lab.LeylineOverrideActive(testutil.BaseTime, cfg))
}

func TestLabState_LeylineOverrideActiveBeforeDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))

	now := testutil.BaseTime.Add(cfg.EmergencyDuration - time.Nanosecond)
	require.True(t, lab.LeylineOverrideActive(now, cfg))
}

func TestLabState_LeylineOverrideInactiveBeforeStart(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	require.False(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(-time.Nanosecond), cfg))
}

func TestLabState_LeylineOverrideInactiveAtDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	require.False(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(cfg.EmergencyDuration), cfg))
}

func TestLabState_LeylineOverrideInactiveAfterDeadline(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	require.False(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(cfg.EmergencyDuration+time.Nanosecond), cfg))
}

func TestLabState_LeylineOverrideInactiveWithoutDeadline(t *testing.T) {
	lab := labAt(75, testutil.BaseTime)
	require.False(t, lab.LeylineOverrideActive(testutil.BaseTime, config.Default()))
}

func TestLabState_LeylineOverrideActiveDoesNotMutateState(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	require.NoError(t, lab.ActivateLeylineOverride(testutil.BaseTime, cfg))
	snapshot := lab

	_ = lab.LeylineOverrideActive(testutil.BaseTime.Add(time.Second), cfg)
	require.Equal(t, snapshot, lab)
}
