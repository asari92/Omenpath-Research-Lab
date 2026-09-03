package domain_test

// Stage 3, checkpoint C: successful OUTBOUND resolution. Tests use explicit
// state fixtures so the behavior under test is not hidden behind StartOutbound.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func observerTestPlane() domain.Plane {
	return domain.Plane{ID: 7, Name: "Innistrad"}
}

func observerTestPortal() domain.Portal {
	portal := testutil.NewPortalBuilder().
		Stable().Energy(100).Decay(0.1).TTL(120 * time.Second).Build()
	portal.ID = 11
	portal.DestinationPlaneID = 7
	return portal
}

func outboundObserver(start, end time.Time) domain.Observer {
	portalID := int64(11)
	return domain.Observer{
		ID:             1,
		Status:         domain.ObserverOutbound,
		ActivePortalID: &portalID,
		PhaseStartedAt: &start,
		PhaseEndsAt:    &end,
		CreatedAt:      testutil.BaseTime,
		UpdatedAt:      start,
	}
}

func TestObserver_OutboundBeforeDeadlineRemainsOutbound(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	snapshot := observer
	plane := observerTestPlane()
	portal := observerTestPortal()

	err := domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(-time.Nanosecond), config.Default())

	require.NoError(t, err)
	require.Equal(t, snapshot, observer)
}

func TestObserver_OutboundAtDeadlineBecomesExploring(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverExploring, observer.Status)
}

func TestObserver_OutboundArrivalUsesDeadlineAsTransitionTime(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(5*time.Second), config.Default()))
	require.Equal(t, end, *observer.PhaseStartedAt)
	require.Equal(t, end, observer.UpdatedAt)
}

func TestObserver_OutboundArrivalSetsCurrentPlane(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.NotNil(t, observer.CurrentPlaneID)
	require.Equal(t, plane.ID, *observer.CurrentPlaneID)
}

func TestObserver_OutboundArrivalClearsActivePortal(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Nil(t, observer.ActivePortalID)
}

func TestObserver_OutboundArrivalStartsTwentySecondResearch(t *testing.T) {
	cfg := config.Default()
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, cfg))
	require.Equal(t, end, *observer.PhaseStartedAt)
	require.Equal(t, end.Add(cfg.ResearchDuration), *observer.PhaseEndsAt)
}

func TestObserver_OutboundArrivalDoesNotExplorePlane(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	before := plane
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, before, plane)
}

func TestObserver_OutboundResolveDoesNotConsumeRandom(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	plane := observerTestPlane()
	portal := observerTestPortal()
	rnd := testutil.NewFakeRandom().QueueInt(13)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, 13, rnd.IntInclusive(5, 15), "resolution must not consume a new duration")
}

func TestObserver_OutboundRejectsMismatchedDestination(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)
	snapshot := observer
	plane := observerTestPlane()
	portal := observerTestPortal()
	portal.DestinationPlaneID = 99

	err := domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, snapshot, observer)
}

func TestObserver_OutboundInvariantViolationIsAtomic(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	valid := outboundObserver(testutil.BaseTime, end)
	cases := []struct {
		name   string
		mutate func(*domain.Observer)
	}{
		{"missing active portal", func(observer *domain.Observer) { observer.ActivePortalID = nil }},
		{"missing phase start", func(observer *domain.Observer) { observer.PhaseStartedAt = nil }},
		{"missing phase end", func(observer *domain.Observer) { observer.PhaseEndsAt = nil }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := valid
			tc.mutate(&observer)
			snapshot := observer
			plane := observerTestPlane()
			portal := observerTestPortal()

			err := domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default())

			require.ErrorIs(t, err, domain.ErrObserverInvariant)
			require.Equal(t, snapshot, observer)
		})
	}
}
