package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestClosePortalWithLabEnergy_ChargesFive(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default()))
	require.Equal(t, 35, lab.EnergyBase)
}

func TestClosePortalWithLabEnergy_AllowsExactFive(t *testing.T) {
	lab := labAt(5, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default()))
	require.Zero(t, lab.EnergyBase)
}

func TestClosePortalWithLabEnergy_UsesRegeneratedEnergy(t *testing.T) {
	lab := labAt(4, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	now := testutil.BaseTime.Add(time.Second)

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, config.Default()))
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestClosePortalWithLabEnergy_RejectsFour(t *testing.T) {
	lab := labAt(4, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default())
	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
}

func TestClosePortalWithLabEnergy_InsufficientLeavesPortalUnchanged(t *testing.T) {
	lab := labAt(4, testutil.BaseTime)
	portal := stage4Portal()
	snapshot := portal
	plane := stage4Plane()

	_ = domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default())
	require.Equal(t, snapshot, portal)
}

func TestClosePortalWithLabEnergy_InsufficientLeavesObserverUnchanged(t *testing.T) {
	lab := labAt(4, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}
	snapshot := append([]domain.Observer(nil), observers...)

	_ = domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, true, config.Default())
	require.Equal(t, snapshot, observers)
}

func TestClosePortalWithLabEnergy_ConfirmationRequiredDoesNotDebit(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, snapshot, lab)
}

func TestClosePortalWithLabEnergy_ConfirmedTransitClosesAndLosesObserver(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, true, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.ObserverLost, observers[0].Status)
}

func TestClosePortalWithLabEnergy_ConfirmedTransitDebitsOnce(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, true, config.Default()))
	require.Equal(t, 5, lab.EnergyBase)
}

func TestClosePortalWithLabEnergy_TerminalPortalDoesNotDebit(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	plane := stage4Plane()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
	require.Equal(t, snapshot, lab)
}

func TestClosePortalWithLabEnergy_StructuralErrorDoesNotDebit(t *testing.T) {
	lab := labAt(10, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, nil, nil, testutil.BaseTime, false, config.Default())
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, snapshot, lab)
}

func TestClosePortalWithLabEnergy_RejectsInvalidLabStateBeforeClose(t *testing.T) {
	lab := labAt(-1, testutil.BaseTime)
	portal := stage4Portal()
	snapshot := portal
	plane := stage4Plane()

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, testutil.BaseTime, false, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, portal)
}
