package domain_test

// Stage 3, checkpoint E: starting RETURNING and resolving a successful
// return. Only this successful transition may explore the Plane.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func returningObserver(start, end time.Time) domain.Observer {
	planeID := int64(7)
	portalID := int64(11)
	return domain.Observer{
		ID:             1,
		Status:         domain.ObserverReturning,
		CurrentPlaneID: &planeID,
		ActivePortalID: &portalID,
		PhaseStartedAt: &start,
		PhaseEndsAt:    &end,
		CreatedAt:      testutil.BaseTime,
		UpdatedAt:      start,
	}
}

func TestObserver_StartReturningFromWaitingReturn(t *testing.T) {
	waitingSince := testutil.BaseTime.Add(30 * time.Second)
	start := testutil.BaseTime.Add(time.Minute)
	observer := waitingObserver(waitingSince)
	rnd := testutil.NewFakeRandom().QueueInt(10)

	err := observer.StartReturning(start, 11, rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.ObserverReturning, observer.Status)
	require.NotNil(t, observer.CurrentPlaneID)
	require.Equal(t, int64(7), *observer.CurrentPlaneID)
	require.NotNil(t, observer.ActivePortalID)
	require.Equal(t, int64(11), *observer.ActivePortalID)
	require.Equal(t, start, *observer.PhaseStartedAt)
	require.Equal(t, start.Add(10*time.Second), *observer.PhaseEndsAt)
	require.Equal(t, start, observer.UpdatedAt)
	require.Equal(t, testutil.BaseTime, observer.CreatedAt)
}

func TestObserver_StartReturningDrawsFreshTransitDuration(t *testing.T) {
	start := testutil.BaseTime.Add(time.Minute)
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	rnd := testutil.NewFakeRandom().QueueInt(15)

	require.NoError(t, observer.StartReturning(start, 11, rnd, config.Default()))
	require.Equal(t, start.Add(15*time.Second), *observer.PhaseEndsAt)
}

func TestObserver_StartReturningDrawsTransitDurationExactlyOnce(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	require.NoError(t, observer.StartReturning(testutil.BaseTime.Add(time.Minute), 11, rnd, config.Default()))
	require.Equal(t, 14, rnd.IntInclusive(5, 15), "only one return duration should be consumed")
}

func TestObserver_StartReturningPreservesCurrentPlaneDuringTransit(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	rnd := testutil.NewFakeRandom().QueueInt(10)

	require.NoError(t, observer.StartReturning(testutil.BaseTime.Add(time.Minute), 11, rnd, config.Default()))
	require.NotNil(t, observer.CurrentPlaneID)
	require.Equal(t, int64(7), *observer.CurrentPlaneID)
}

func TestObserver_StartReturningRejectsInvalidStates(t *testing.T) {
	for _, status := range []domain.ObserverStatus{
		domain.ObserverAvailable,
		domain.ObserverOutbound,
		domain.ObserverExploring,
		domain.ObserverReturning,
	} {
		t.Run(string(status), func(t *testing.T) {
			observer := domain.Observer{ID: 1, Status: status, UpdatedAt: testutil.BaseTime}
			snapshot := observer

			err := observer.StartReturning(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

			require.ErrorIs(t, err, domain.ErrObserverNotWaitingReturn)
			require.Equal(t, snapshot, observer)
		})
	}

	t.Run("lost", func(t *testing.T) {
		observer := domain.Observer{ID: 1, Status: domain.ObserverLost, UpdatedAt: testutil.BaseTime}
		snapshot := observer

		err := observer.StartReturning(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

		require.ErrorIs(t, err, domain.ErrObserverLost)
		require.Equal(t, snapshot, observer)
	})
}

func TestObserver_StartReturningFailureIsAtomic(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	observer.CurrentPlaneID = nil
	snapshot := observer

	err := observer.StartReturning(testutil.BaseTime.Add(time.Minute), 11, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, snapshot, observer)
}

func TestObserver_ReturningBeforeDeadlineRemainsReturning(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	snapshot := observer
	plane := observerTestPlane()
	portal := observerTestPortal()

	err := domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(-time.Nanosecond), config.Default())

	require.NoError(t, err)
	require.Equal(t, snapshot, observer)
	require.False(t, plane.Explored)
}

func TestObserver_ReturningAtDeadlineBecomesAvailable(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, domain.ObserverAvailable, observer.Status)
}

func TestObserver_ReturnUsesDeadlineAsTransitionTime(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(time.Minute), config.Default()))
	require.Equal(t, end, observer.UpdatedAt)
}

func TestObserver_ReturnClearsCurrentPlaneAndActivePortal(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Nil(t, observer.CurrentPlaneID)
	require.Nil(t, observer.ActivePortalID)
	require.Nil(t, observer.PhaseStartedAt)
	require.Nil(t, observer.PhaseEndsAt)
}

func TestObserver_SuccessfulReturnExploresPlane(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.True(t, plane.Explored)
}

func TestObserver_SuccessfulReturnSetsExploredAtToArrival(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end.Add(time.Minute), config.Default()))
	require.NotNil(t, plane.ExploredAt)
	require.Equal(t, end, *plane.ExploredAt)
}

func TestObserver_ReturnToAlreadyExploredPlanePreservesOriginalExploredAt(t *testing.T) {
	firstExploredAt := testutil.BaseTime.Add(40 * time.Second)
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	plane.Explored = true
	plane.ExploredAt = &firstExploredAt
	portal := observerTestPortal()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.True(t, plane.Explored)
	require.Equal(t, firstExploredAt, *plane.ExploredAt)
}

func TestObserver_ReturnResolveDoesNotConsumeRandom(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	observer := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	plane := observerTestPlane()
	portal := observerTestPortal()
	rnd := testutil.NewFakeRandom().QueueInt(12)

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, &portal, end, config.Default()))
	require.Equal(t, 12, rnd.IntInclusive(5, 15), "return resolution must not consume randomness")
}

func TestObserver_ReturningInvariantViolationIsAtomic(t *testing.T) {
	end := testutil.BaseTime.Add(70 * time.Second)
	valid := returningObserver(testutil.BaseTime.Add(time.Minute), end)
	cases := []struct {
		name   string
		mutate func(*domain.Observer)
	}{
		{"missing current plane", func(observer *domain.Observer) { observer.CurrentPlaneID = nil }},
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
