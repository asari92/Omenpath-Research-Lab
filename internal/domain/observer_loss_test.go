package domain_test

// Stage 3, checkpoint F: a Portal terminal transition strictly before a
// transit deadline makes the Observer LOST. Tests cover both directions and
// both CLOSED/COLLAPSED outcomes without coupling Portal mutations to Observer.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func terminalObserverPortal(status domain.PortalStatus, reason domain.TerminationReason, closedAt time.Time) domain.Portal {
	portal := observerTestPortal()
	portal.Status = status
	portal.TerminationReason = reason
	portal.ClosedAt = &closedAt
	portal.UpdatedAt = closedAt
	return portal
}

func TestObserver_OutboundLostWhenPortalClosesBeforeTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusClosed, domain.TerminationNaturalClose, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverLost, observer.Status)
}

func TestObserver_OutboundLostWhenPortalCollapsesBeforeTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverLost, observer.Status)
}

func TestObserver_ReturningLostWhenPortalClosesBeforeTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusClosed, domain.TerminationManualClose, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverLost, observer.Status)
}

func TestObserver_ReturningLostWhenPortalCollapsesBeforeTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationInstability, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverLost, observer.Status)
}

func TestObserver_LossUsesPortalClosedAtAsTransitionTime(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	closedAt := testutil.BaseTime.Add(6 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusClosed, domain.TerminationNaturalClose, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(time.Minute), config.Default()))
	require.Equal(t, closedAt, observer.UpdatedAt)
}

func TestObserver_LossClearsActiveState(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverLost, observer.Status)
	require.Nil(t, observer.CurrentPlaneID)
	require.Nil(t, observer.ActivePortalID)
	require.Nil(t, observer.PhaseStartedAt)
	require.Nil(t, observer.PhaseEndsAt)
}

func TestObserver_LossDoesNotExplorePlane(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	closedAt := end.Add(-time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	before := plane
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationInstability, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, before, plane)
}

func TestObserver_LostCannotBeRevivedByResolve(t *testing.T) {
	observer := domain.Observer{ID: 1, Status: domain.ObserverLost, UpdatedAt: testutil.BaseTime}
	snapshot := observer
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, testutil.BaseTime.Add(time.Hour), config.Default()))
	require.Equal(t, snapshot, observer)
}

func TestObserver_LostCannotStartOutbound(t *testing.T) {
	observer := domain.Observer{ID: 1, Status: domain.ObserverLost, UpdatedAt: testutil.BaseTime}
	snapshot := observer

	err := observer.StartOutbound(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverLost)
	require.Equal(t, snapshot, observer)
}

func TestObserver_LostCannotStartReturning(t *testing.T) {
	observer := domain.Observer{ID: 1, Status: domain.ObserverLost, UpdatedAt: testutil.BaseTime}
	snapshot := observer

	err := observer.StartReturning(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverLost)
	require.Equal(t, snapshot, observer)
}

func TestObserver_TransitRejectsTerminalPortalWithoutClosedAt(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	snapshot := observer
	plane := observerTestPlane()
	portal := observerTestPortal()
	portal.Status = domain.PortalStatusClosed

	err := domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, snapshot, observer)
}
