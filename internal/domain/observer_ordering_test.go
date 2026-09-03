package domain_test

// Stage 3, checkpoint G: ordering, exact ties, and multi-phase catch-up.
// The strict wording "before transit ends" makes equality a success.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestObserver_OutboundSucceedsWhenPortalClosesExactlyAtTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusClosed, domain.TerminationNaturalClose, end)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverExploring, observer.Status)
	require.Equal(t, end, *observer.PhaseStartedAt)
}

func TestObserver_ReturningSucceedsWhenPortalClosesExactlyAtTransitEnd(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusClosed, domain.TerminationManualClose, end)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverAvailable, observer.Status)
	require.True(t, plane.Explored)
	require.Equal(t, end, *plane.ExploredAt)
}

func TestObserver_PortalClosingAfterOutboundEndDoesNotRetroactivelyLoseObserver(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	closedAt := end.Add(2 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationInstability, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(15*time.Second), config.Default()))
	require.Equal(t, domain.ObserverExploring, observer.Status)
	require.Equal(t, end, *observer.PhaseStartedAt)
}

func TestObserver_PortalClosingAfterReturnEndDoesNotRetroactivelyLoseObserver(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	closedAt := end.Add(2 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := terminalObserverPortal(domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, closedAt)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, closedAt.Add(time.Minute), config.Default()))
	require.Equal(t, domain.ObserverAvailable, observer.Status)
	require.True(t, plane.Explored)
}

func TestObserver_CatchUpOutboundThroughResearchEndsWaitingReturn(t *testing.T) {
	transitEnd := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, transitEnd)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(
		&observer, &plane, &portal, testutil.BaseTime.Add(time.Hour), config.Default(),
	))
	require.Equal(t, domain.ObserverWaitingReturn, observer.Status)
	require.Equal(t, transitEnd.Add(config.Default().ResearchDuration), *observer.PhaseStartedAt)
	require.Nil(t, observer.PhaseEndsAt)
	require.False(t, plane.Explored)
}

func TestObserver_CatchUpUsesEffectiveTransitionTimestamps(t *testing.T) {
	transitEnd := testutil.BaseTime.Add(10 * time.Second)
	researchEnd := transitEnd.Add(config.Default().ResearchDuration)
	observer := outboundObserver(testutil.BaseTime, transitEnd)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(
		&observer, &plane, &portal, testutil.BaseTime.Add(time.Hour), config.Default(),
	))
	require.Equal(t, researchEnd, *observer.PhaseStartedAt)
	require.Equal(t, researchEnd, observer.UpdatedAt)
}

func TestObserver_RepeatedResolveIsIdempotent(t *testing.T) {
	transitEnd := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, transitEnd)
	plane := observerTestPlane()
	portal := observerTestPortal()
	now := testutil.BaseTime.Add(time.Hour)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, now, config.Default()))
	observerSnapshot := observer
	planeSnapshot := plane

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, now.Add(time.Hour), config.Default()))
	require.Equal(t, observerSnapshot, observer)
	require.Equal(t, planeSnapshot, plane)
}
