package domain_test

// Stage 3, checkpoint D: deterministic research completion and the
// WAITING_RETURN state. Plane exploration must remain unchanged.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func exploringObserver(start, end time.Time) domain.Observer {
	planeID := int64(7)
	return domain.Observer{
		ID:             1,
		Status:         domain.ObserverExploring,
		CurrentPlaneID: &planeID,
		PhaseStartedAt: &start,
		PhaseEndsAt:    &end,
		CreatedAt:      testutil.BaseTime,
		UpdatedAt:      start,
	}
}

func waitingObserver(since time.Time) domain.Observer {
	planeID := int64(7)
	return domain.Observer{
		ID:             1,
		Status:         domain.ObserverWaitingReturn,
		CurrentPlaneID: &planeID,
		PhaseStartedAt: &since,
		CreatedAt:      testutil.BaseTime,
		UpdatedAt:      since,
	}
}

func TestObserver_ExploringBeforeDeadlineRemainsExploring(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	snapshot := observer
	plane := observerTestPlane()

	err := domain.ResolveObserverLifecycle(&observer, &plane, nil, end.Add(-time.Nanosecond), config.Default())

	require.NoError(t, err)
	require.Equal(t, snapshot, observer)
}

func TestObserver_ResearchCompletesAtDeadline(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	plane := observerTestPlane()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, end, config.Default()))
	require.Equal(t, domain.ObserverWaitingReturn, observer.Status)
}

func TestObserver_ResearchCompletionUsesDeadlineAsTransitionTime(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	plane := observerTestPlane()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, end.Add(time.Minute), config.Default()))
	require.Equal(t, end, *observer.PhaseStartedAt)
	require.Equal(t, end, observer.UpdatedAt)
}

func TestObserver_ResearchCompletionBecomesWaitingReturn(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	plane := observerTestPlane()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, end, config.Default()))
	require.Equal(t, domain.ObserverWaitingReturn, observer.Status)
	require.NotNil(t, observer.CurrentPlaneID)
	require.Equal(t, plane.ID, *observer.CurrentPlaneID)
	require.Nil(t, observer.ActivePortalID)
	require.Nil(t, observer.PhaseEndsAt)
}

func TestObserver_WaitingReturnStartsAtResearchCompletion(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	plane := observerTestPlane()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, end.Add(time.Second), config.Default()))
	require.Equal(t, end, *observer.PhaseStartedAt)
}

func TestObserver_ResearchCompletionDoesNotExplorePlane(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	observer := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	plane := observerTestPlane()
	before := plane

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, end, config.Default()))
	require.Equal(t, before, plane)
}

func TestObserver_WaitingReturnHasNoAutomaticEnd(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	snapshot := observer
	plane := observerTestPlane()

	require.NoError(t, domain.ResolveObserverLifecycle(&observer, &plane, nil, testutil.BaseTime.Add(time.Hour), config.Default()))
	require.Equal(t, snapshot, observer)
}

func TestObserver_ExploringInvariantViolationIsAtomic(t *testing.T) {
	end := testutil.BaseTime.Add(30 * time.Second)
	valid := exploringObserver(testutil.BaseTime.Add(10*time.Second), end)
	cases := []struct {
		name   string
		mutate func(*domain.Observer)
	}{
		{"missing current plane", func(observer *domain.Observer) { observer.CurrentPlaneID = nil }},
		{"missing phase start", func(observer *domain.Observer) { observer.PhaseStartedAt = nil }},
		{"missing phase end", func(observer *domain.Observer) { observer.PhaseEndsAt = nil }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			observer := valid
			tc.mutate(&observer)
			snapshot := observer
			plane := observerTestPlane()

			err := domain.ResolveObserverLifecycle(&observer, &plane, nil, end, config.Default())

			require.ErrorIs(t, err, domain.ErrObserverInvariant)
			require.Equal(t, snapshot, observer)
		})
	}
}

func TestObserver_WaitingReturnRequiresCurrentPlane(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	observer.CurrentPlaneID = nil
	snapshot := observer
	plane := observerTestPlane()

	err := domain.ResolveObserverLifecycle(&observer, &plane, nil, testutil.BaseTime.Add(time.Hour), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, snapshot, observer)
}
