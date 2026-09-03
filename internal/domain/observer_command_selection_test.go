package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestAvailableObserverIndex_SelectsLowestAvailableID(t *testing.T) {
	observers := []domain.Observer{
		domain.NewObserver(8, testutil.BaseTime),
		domain.NewObserver(2, testutil.BaseTime),
		domain.NewObserver(5, testutil.BaseTime),
	}

	index, ok := domain.AvailableObserverIndex(observers)

	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestAvailableObserverIndex_IgnoresNonAvailableObservers(t *testing.T) {
	observers := []domain.Observer{
		{ID: 1, Status: domain.ObserverLost},
		domain.NewObserver(7, testutil.BaseTime),
		{ID: 3, Status: domain.ObserverExploring},
	}

	index, ok := domain.AvailableObserverIndex(observers)

	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestAvailableObserverIndex_ReturnsNoneWhenUnavailable(t *testing.T) {
	observers := []domain.Observer{
		{ID: 1, Status: domain.ObserverLost},
		{ID: 2, Status: domain.ObserverExploring},
	}

	index, ok := domain.AvailableObserverIndex(observers)

	require.False(t, ok)
	require.Equal(t, -1, index)
}

func TestLongestWaitingObserverIndex_SelectsEarliestWaitingTimestamp(t *testing.T) {
	newer := waitingObserver(testutil.BaseTime.Add(30 * time.Second))
	newer.ID = 1
	older := waitingObserver(testutil.BaseTime.Add(20 * time.Second))
	older.ID = 9

	index, ok, err := domain.LongestWaitingObserverIndex([]domain.Observer{newer, older}, 7)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestLongestWaitingObserverIndex_FiltersByDestinationPlane(t *testing.T) {
	otherPlaneID := int64(99)
	otherPlane := waitingObserver(testutil.BaseTime.Add(10 * time.Second))
	otherPlane.ID = 1
	otherPlane.CurrentPlaneID = &otherPlaneID
	matching := waitingObserver(testutil.BaseTime.Add(20 * time.Second))
	matching.ID = 8

	index, ok, err := domain.LongestWaitingObserverIndex([]domain.Observer{otherPlane, matching}, 7)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestLongestWaitingObserverIndex_BreaksExactTieByLowestID(t *testing.T) {
	since := testutil.BaseTime.Add(20 * time.Second)
	higherID := waitingObserver(since)
	higherID.ID = 8
	lowerID := waitingObserver(since)
	lowerID.ID = 2

	index, ok, err := domain.LongestWaitingObserverIndex([]domain.Observer{higherID, lowerID}, 7)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 1, index)
}

func TestLongestWaitingObserverIndex_RejectsMissingWaitingTimestamp(t *testing.T) {
	observer := waitingObserver(testutil.BaseTime)
	observer.PhaseStartedAt = nil

	index, ok, err := domain.LongestWaitingObserverIndex([]domain.Observer{observer}, 7)

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.False(t, ok)
	require.Equal(t, -1, index)
}

func TestActiveTransitObserverIndex_FindsOutbound(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)

	index, ok, err := domain.ActiveTransitObserverIndex([]domain.Observer{observer}, 11, testutil.BaseTime)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 0, index)
}

func TestActiveTransitObserverIndex_FindsReturning(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := returningObserver(testutil.BaseTime, end)

	index, ok, err := domain.ActiveTransitObserverIndex([]domain.Observer{observer}, 11, testutil.BaseTime)

	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 0, index)
}

func TestActiveTransitObserverIndex_IgnoresOtherPortals(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)

	index, ok, err := domain.ActiveTransitObserverIndex([]domain.Observer{observer}, 12, testutil.BaseTime)

	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, -1, index)
}

func TestActiveTransitObserverIndex_ReturnsNoneWhenIdle(t *testing.T) {
	observers := []domain.Observer{
		domain.NewObserver(1, testutil.BaseTime),
		waitingObserver(testutil.BaseTime),
	}

	index, ok, err := domain.ActiveTransitObserverIndex(observers, 11, testutil.BaseTime)

	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, -1, index)
}

func TestActiveTransitObserverIndex_RejectsMultipleTransitsForSamePortal(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	first := outboundObserver(testutil.BaseTime, end)
	second := returningObserver(testutil.BaseTime, end)
	second.ID = 2

	index, ok, err := domain.ActiveTransitObserverIndex([]domain.Observer{first, second}, 11, testutil.BaseTime)

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.False(t, ok)
	require.Equal(t, -1, index)
}

func TestActiveTransitObserverIndex_RejectsStaleTransitAtDeadline(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)

	index, ok, err := domain.ActiveTransitObserverIndex([]domain.Observer{observer}, 11, end)

	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.False(t, ok)
	require.Equal(t, -1, index)
}
