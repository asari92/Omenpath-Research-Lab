package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func overrideLab(energy int, collapseAt time.Time, cfg config.Config) domain.LabState {
	deadline := collapseAt.Add(cfg.EmergencyDuration)
	return domain.LabState{
		EnergyBase:           energy,
		EnergyBaseAt:         collapseAt,
		LeylineOverrideUntil: &deadline,
	}
}

func TestClosePortalWithLabEnergy_ActiveOverrideCostsZero(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()
	now := testutil.BaseTime.Add(5 * time.Second)
	energyBefore := lab.CurrentEnergy(now, cfg)

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg))
	require.Equal(t, energyBefore, lab.CurrentEnergy(now, cfg))
}

func TestClosePortalWithLabEnergy_ActiveOverrideAllowsZeroEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, cfg))
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
}

func TestClosePortalWithLabEnergy_ActiveOverrideDoesNotRebaseEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime.Add(5*time.Second), false, cfg))
	require.Equal(t, snapshot, lab)
}

func TestClosePortalWithLabEnergy_ActiveOverridePreservesDeadline(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	deadline := lab.LeylineOverrideUntil
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, cfg))
	require.Same(t, deadline, lab.LeylineOverrideUntil)
}

func TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresCreatureConfirmation(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	portal.CreaturesInitial = 1
	plane := stage4Plane()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, cfg)
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
}

func TestClosePortalWithLabEnergy_ActiveOverrideStillRequiresTransitConfirmation(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, cfg)
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
}

func TestClosePortalWithLabEnergy_ActiveOverrideConfirmedTransitLosesObserver(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, true, cfg))
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.ObserverLost, observers[0].Status)
}

func TestClosePortalWithLabEnergy_ActiveOverrideTerminalPortalStillRejects(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	plane := stage4Plane()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, cfg)
	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestClosePortalWithLabEnergy_AtOverrideDeadlineChargesNormalCost(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg))
	expected := int(cfg.EmergencyDuration/time.Second)*cfg.LabRegenPerSec - cfg.CloseCost
	require.Equal(t, expected, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestClosePortalWithLabEnergy_ExpiredOverrideInsufficientIsAtomic(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = 0
	lab := overrideLab(0, testutil.BaseTime, cfg)
	labBefore := lab
	portal := stage4Portal()
	portalBefore := portal
	plane := stage4Plane()
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg)

	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}

func TestClosePortalWithLabEnergy_ExpiredDeadlineIsNotClearedByReadOrCommand(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = 0
	lab := overrideLab(0, testutil.BaseTime, cfg)
	deadline := lab.LeylineOverrideUntil
	portal := stage4Portal()
	plane := stage4Plane()
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	require.False(t, lab.LeylineOverrideActive(now, cfg))
	_ = domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg)
	require.Same(t, deadline, lab.LeylineOverrideUntil)
}
