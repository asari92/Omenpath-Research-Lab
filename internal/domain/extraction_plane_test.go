package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func waitingObserverInPlane(id, planeID int64, since time.Time) domain.Observer {
	return domain.Observer{ID: id, Status: domain.ObserverWaitingReturn, CurrentPlaneID: &planeID,
		PhaseStartedAt: &since, CreatedAt: testutil.BaseTime, UpdatedAt: since}
}

func TestExtractionPlaneEligible_ReturnsTrueForWaitingObserver(t *testing.T) {
	ok, err := domain.ExtractionPlaneEligible([]domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.True(t, ok)
}
func TestExtractionPlaneEligible_ReturnsFalseForEmptyRoster(t *testing.T) {
	ok, err := domain.ExtractionPlaneEligible(nil, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.False(t, ok)
}
func TestExtractionPlaneEligible_ReturnsFalseWithoutWaitingObservers(t *testing.T) {
	ok, err := domain.ExtractionPlaneEligible([]domain.Observer{domain.NewObserver(1, testutil.BaseTime)}, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.False(t, ok)
}
func TestExtractionPlaneEligible_FiltersBySelectedPlane(t *testing.T) {
	ok, err := domain.ExtractionPlaneEligible([]domain.Observer{waitingObserverInPlane(1, 8, testutil.BaseTime)}, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.False(t, ok)
}
func TestExtractionPlaneEligible_AcceptsMultipleWaitingObservers(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	ok, err := domain.ExtractionPlaneEligible(os, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.True(t, ok)
}

func eligibleIgnoresStatus(t *testing.T, status domain.ObserverStatus) {
	t.Helper()
	p := int64(7)
	started := testutil.BaseTime
	end := started.Add(time.Minute)
	o := domain.Observer{ID: 1, Status: status, CreatedAt: started, UpdatedAt: started}
	switch status {
	case domain.ObserverExploring:
		o.CurrentPlaneID, o.PhaseStartedAt, o.PhaseEndsAt = &p, &started, &end
	case domain.ObserverReturning:
		portalID := int64(9)
		o.CurrentPlaneID, o.ActivePortalID, o.PhaseStartedAt, o.PhaseEndsAt = &p, &portalID, &started, &end
	}
	ok, err := domain.ExtractionPlaneEligible([]domain.Observer{o}, 7, started)
	require.NoError(t, err)
	require.False(t, ok)
}
func TestExtractionPlaneEligible_IgnoresAvailableObserver(t *testing.T) {
	eligibleIgnoresStatus(t, domain.ObserverAvailable)
}
func TestExtractionPlaneEligible_IgnoresExploringObserver(t *testing.T) {
	eligibleIgnoresStatus(t, domain.ObserverExploring)
}
func TestExtractionPlaneEligible_IgnoresReturningObserver(t *testing.T) {
	eligibleIgnoresStatus(t, domain.ObserverReturning)
}
func TestExtractionPlaneEligible_IgnoresLostObserver(t *testing.T) {
	eligibleIgnoresStatus(t, domain.ObserverLost)
}

func TestExtractionPlaneEligible_RejectsWaitingWithoutPlane(t *testing.T) {
	s := testutil.BaseTime
	o := domain.Observer{ID: 1, Status: domain.ObserverWaitingReturn, PhaseStartedAt: &s}
	_, err := domain.ExtractionPlaneEligible([]domain.Observer{o}, 7, s)
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
}
func TestExtractionPlaneEligible_RejectsWaitingWithoutTimestamp(t *testing.T) {
	p := int64(7)
	o := domain.Observer{ID: 1, Status: domain.ObserverWaitingReturn, CurrentPlaneID: &p}
	_, err := domain.ExtractionPlaneEligible([]domain.Observer{o}, 7, testutil.BaseTime)
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
}
func TestExtractionPlaneEligible_RejectsDuplicateObserverIDs(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime), waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.ExtractionPlaneEligible(os, 7, testutil.BaseTime)
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
}
func TestExtractionPlaneEligible_RejectsStaleTransitState(t *testing.T) {
	p, portalID := int64(7), int64(9)
	start := testutil.BaseTime.Add(-time.Minute)
	end := testutil.BaseTime
	o := domain.Observer{ID: 1, Status: domain.ObserverReturning, CurrentPlaneID: &p, ActivePortalID: &portalID, PhaseStartedAt: &start, PhaseEndsAt: &end}
	_, err := domain.ExtractionPlaneEligible([]domain.Observer{o}, 7, testutil.BaseTime)
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
}
func TestExtractionPlaneEligible_DoesNotMutateOrReorderRoster(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(2, 7, testutil.BaseTime), waitingObserverInPlane(1, 7, testutil.BaseTime)}
	before := append([]domain.Observer(nil), os...)
	_, err := domain.ExtractionPlaneEligible(os, 7, testutil.BaseTime)
	require.NoError(t, err)
	require.Equal(t, before, os)
}
func TestObserverCommandValidation_RefactorPreservesExistingErrors(t *testing.T) {
	p := stage4Portal()
	plane := stage4Plane()
	o := domain.NewObserver(1, testutil.BaseTime)
	os := []domain.Observer{o, o}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
}
