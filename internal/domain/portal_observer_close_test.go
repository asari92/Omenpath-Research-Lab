package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestClosePortalWithObservers_ActiveOutboundRequiresConfirmation(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), false, config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, domain.PortalStatusOpen, portal.Status)
	require.Equal(t, domain.ObserverOutbound, observers[0].Status)
}

func TestClosePortalWithObservers_ActiveReturningRequiresConfirmation(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := returningObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), false, config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, domain.PortalStatusOpen, portal.Status)
	require.Equal(t, domain.ObserverReturning, observers[0].Status)
}

func TestClosePortalWithObservers_ConfirmedOutboundBecomesLost(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), true, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.TerminationManualClose, portal.TerminationReason)
	require.Equal(t, domain.ObserverLost, observers[0].Status)
}

func TestClosePortalWithObservers_ConfirmedReturningBecomesLost(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := returningObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), true, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.ObserverLost, observers[0].Status)
}

func TestClosePortalWithObservers_ConfirmedCloseUsesNowForBothTransitions(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}
	now := testutil.BaseTime.Add(5 * time.Second)

	require.NoError(t, domain.ClosePortalWithObservers(&portal, &plane, observers, now, true, config.Default()))

	require.NotNil(t, portal.ClosedAt)
	require.Equal(t, now, *portal.ClosedAt)
	require.Equal(t, now, portal.UpdatedAt)
	require.Equal(t, now, observers[0].UpdatedAt)
}

func TestClosePortalWithObservers_LossClearsObserverCurrentState(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := returningObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	require.NoError(t, domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), true, config.Default()))

	require.Nil(t, observers[0].CurrentPlaneID)
	require.Nil(t, observers[0].ActivePortalID)
	require.Nil(t, observers[0].PhaseStartedAt)
	require.Nil(t, observers[0].PhaseEndsAt)
}

func TestClosePortalWithObservers_LossDoesNotExplorePlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	planeBefore := plane
	observer := returningObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}

	require.NoError(t, domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), true, config.Default()))

	require.Equal(t, planeBefore, plane)
}

func TestClosePortalWithObservers_CreaturesAndTransitUseSingleConfirmation(t *testing.T) {
	portal := stage4Portal()
	portal.CreaturesInitial = 2
	plane := stage4Plane()
	observer := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}
	now := testutil.BaseTime

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, now, false, config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)

	require.NoError(t, domain.ClosePortalWithObservers(&portal, &plane, observers, now, true, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.ObserverLost, observers[0].Status)
}

func TestClosePortalWithObservers_NoTransitDelegatesToPortalClose(t *testing.T) {
	portal := stage4Portal()
	portal.CreaturesInitial = 1
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime, false, config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, domain.PortalStatusOpen, portal.Status)

	require.NoError(t, domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime, true, config.Default()))
	require.Equal(t, domain.PortalStatusClosed, portal.Status)
	require.Equal(t, domain.ObserverAvailable, observers[0].Status)
}

func TestClosePortalWithObservers_RejectsTerminalPortal(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	portalBefore := portal

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime, true, config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
	require.Equal(t, portalBefore, portal)
}

func TestClosePortalWithObservers_RejectsMultipleActiveTransits(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	end := testutil.BaseTime.Add(10 * time.Second)
	first := outboundObserver(testutil.BaseTime, end)
	second := returningObserver(testutil.BaseTime, end)
	second.ID = 2
	observers := []domain.Observer{first, second}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime, true, config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
}

func TestClosePortalWithObservers_RejectsStaleTransitBeforeMutation(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	observers := []domain.Observer{observer}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, end, true, config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
}

func TestClosePortalWithObservers_RejectionIsAtomic(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := returningObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{observer}
	portalBefore := portal
	planeBefore := plane
	observersBefore := append([]domain.Observer(nil), observers...)

	err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), false, config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, planeBefore, plane)
	require.Equal(t, observersBefore, observers)
}
