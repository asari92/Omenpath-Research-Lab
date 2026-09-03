package domain_test

// Stage 3, checkpoint B: starting an already-authorized OUTBOUND transit.
// Portal admission policy belongs to Stage 4 and is intentionally absent.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestObserver_StartOutboundTransitionsFromAvailable(t *testing.T) {
	observer := domain.NewObserver(1, testutil.BaseTime)
	rnd := testutil.NewFakeRandom().QueueInt(10)

	err := observer.StartOutbound(testutil.BaseTime, 11, rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, domain.ObserverOutbound, observer.Status)
	require.Nil(t, observer.CurrentPlaneID)
	require.NotNil(t, observer.ActivePortalID)
	require.Equal(t, int64(11), *observer.ActivePortalID)
	require.NotNil(t, observer.PhaseStartedAt)
	require.Equal(t, testutil.BaseTime, *observer.PhaseStartedAt)
	require.NotNil(t, observer.PhaseEndsAt)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *observer.PhaseEndsAt)
	require.Equal(t, testutil.BaseTime, observer.UpdatedAt)
	require.Equal(t, testutil.BaseTime, observer.CreatedAt)
}

func TestObserver_StartOutboundTransitMinimumFiveSeconds(t *testing.T) {
	cfg := config.Default()
	observer := domain.NewObserver(1, testutil.BaseTime)
	rnd := testutil.NewFakeRandom().QueueInt(int(cfg.ObserverTransitMin.Seconds()))

	require.NoError(t, observer.StartOutbound(testutil.BaseTime, 11, rnd, cfg))
	require.Equal(t, testutil.BaseTime.Add(cfg.ObserverTransitMin), *observer.PhaseEndsAt)
}

func TestObserver_StartOutboundTransitMaximumFifteenSeconds(t *testing.T) {
	cfg := config.Default()
	observer := domain.NewObserver(1, testutil.BaseTime)
	rnd := testutil.NewFakeRandom().QueueInt(int(cfg.ObserverTransitMax.Seconds()))

	require.NoError(t, observer.StartOutbound(testutil.BaseTime, 11, rnd, cfg))
	require.Equal(t, testutil.BaseTime.Add(cfg.ObserverTransitMax), *observer.PhaseEndsAt)
}

func TestObserver_StartOutboundDrawsTransitDurationExactlyOnce(t *testing.T) {
	observer := domain.NewObserver(1, testutil.BaseTime)
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	require.NoError(t, observer.StartOutbound(testutil.BaseTime, 11, rnd, config.Default()))
	require.Equal(t, 14, rnd.IntInclusive(5, 15), "only the first queued duration should be consumed")
}

func TestObserver_StartOutboundDoesNotExplorePlane(t *testing.T) {
	plane := domain.Plane{ID: 7}
	before := plane
	observer := domain.NewObserver(1, testutil.BaseTime)
	rnd := testutil.NewFakeRandom().QueueInt(10)

	require.NoError(t, observer.StartOutbound(testutil.BaseTime, 11, rnd, config.Default()))
	require.Equal(t, before, plane)
	require.False(t, plane.Explored)
	require.Nil(t, plane.ExploredAt)
}

func TestObserver_StartOutboundRejectsNonAvailableState(t *testing.T) {
	for _, status := range []domain.ObserverStatus{
		domain.ObserverOutbound,
		domain.ObserverExploring,
		domain.ObserverWaitingReturn,
		domain.ObserverReturning,
	} {
		t.Run(string(status), func(t *testing.T) {
			observer := domain.Observer{ID: 1, Status: status, UpdatedAt: testutil.BaseTime}
			snapshot := observer

			err := observer.StartOutbound(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

			require.ErrorIs(t, err, domain.ErrObserverNotAvailable)
			require.Equal(t, snapshot, observer)
		})
	}
}

func TestObserver_StartOutboundRejectsLostObserver(t *testing.T) {
	observer := domain.Observer{ID: 1, Status: domain.ObserverLost, UpdatedAt: testutil.BaseTime}
	snapshot := observer

	err := observer.StartOutbound(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverLost)
	require.Equal(t, snapshot, observer)
}

func TestObserver_StartOutboundFailureIsAtomic(t *testing.T) {
	planeID := int64(7)
	phaseStart := testutil.BaseTime.Add(-20 * time.Second)
	phaseEnd := testutil.BaseTime
	observer := domain.Observer{
		ID:             1,
		Status:         domain.ObserverExploring,
		CurrentPlaneID: &planeID,
		PhaseStartedAt: &phaseStart,
		PhaseEndsAt:    &phaseEnd,
		CreatedAt:      testutil.BaseTime.Add(-time.Hour),
		UpdatedAt:      phaseStart,
	}
	snapshot := observer

	err := observer.StartOutbound(testutil.BaseTime, 11, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrObserverNotAvailable)
	require.Equal(t, snapshot, observer)
}
