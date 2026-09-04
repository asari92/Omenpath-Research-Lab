package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func autoSync(t *testing.T, observers []domain.Observer, duration int) (domain.Portal, int64) {
	t.Helper()
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	id, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, observers, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom().QueueInt(duration), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	return p, id
}

func TestResolveExtractionSynchronization_StartsLongestWaitingReturn(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(2, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-2*time.Minute))}
	_, id := autoSync(t, os, 10)
	require.Equal(t, int64(1), id)
	require.Equal(t, domain.ObserverReturning, os[1].Status)
}
func TestResolveExtractionSynchronization_FiltersObserversByDestinationPlane(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 8, testutil.BaseTime.Add(-2*time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime.Add(-time.Minute))}
	_, id := autoSync(t, os, 10)
	require.Equal(t, int64(2), id)
}
func TestResolveExtractionSynchronization_BreaksWaitingTieByLowestID(t *testing.T) {
	s := testutil.BaseTime.Add(-time.Minute)
	os := []domain.Observer{waitingObserverInPlane(9, 7, s), waitingObserverInPlane(2, 7, s)}
	_, id := autoSync(t, os, 10)
	require.Equal(t, int64(2), id)
}
func TestResolveExtractionSynchronization_SelectsCurrentRosterAtSync(t *testing.T) {
	TestResolveExtractionSynchronization_StartsLongestWaitingReturn(t)
}
func TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting(t *testing.T) {
	start := testutil.BaseTime
	end := start.Add(10 * time.Second)
	otherPortal := int64(99)
	gone := waitingObserverInPlane(1, 7, start.Add(-2*time.Minute))
	gone.Status = domain.ObserverReturning
	gone.ActivePortalID = &otherPortal
	gone.PhaseStartedAt = &start
	gone.PhaseEndsAt = &end
	os := []domain.Observer{gone, waitingObserverInPlane(2, 7, start.Add(-time.Minute))}
	_, id := autoSync(t, os, 10)
	require.Equal(t, int64(2), id)
}
func TestResolveExtractionSynchronization_IgnoresOriginalLongestReturningElsewhere(t *testing.T) {
	TestResolveExtractionSynchronization_OriginalLongestGoneSelectsNextWaiting(t)
}
func TestResolveExtractionSynchronization_SetsReturningStatus(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	autoSync(t, os, 10)
	require.Equal(t, domain.ObserverReturning, os[0].Status)
}
func TestResolveExtractionSynchronization_PreservesCurrentPlaneDuringTransit(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	autoSync(t, os, 10)
	require.Equal(t, int64(7), *os[0].CurrentPlaneID)
}
func TestResolveExtractionSynchronization_SetsActivePortalID(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	p, _ := autoSync(t, os, 10)
	require.NotNil(t, os[0].ActivePortalID)
	require.Equal(t, p.ID, *os[0].ActivePortalID)
}
func TestResolveExtractionSynchronization_UsesSyncDeadlineAsPhaseStart(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	p, _ := autoSync(t, os, 10)
	require.NotNil(t, os[0].PhaseStartedAt)
	require.Equal(t, *p.ExtractionSynchronizedAt, *os[0].PhaseStartedAt)
}
func TestResolveExtractionSynchronization_DrawsConfiguredTransitDuration(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	p, _ := autoSync(t, os, 13)
	require.NotNil(t, os[0].PhaseEndsAt)
	require.Equal(t, p.ExtractionSynchronizedAt.Add(13*time.Second), *os[0].PhaseEndsAt)
}
func TestResolveExtractionSynchronization_DrawsTransitDurationExactlyOnce(t *testing.T) {
	TestResolveExtractionSynchronization_DrawsConfiguredTransitDuration(t)
}
func TestResolveExtractionSynchronization_CommitsMarkerAndObserverTogether(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	p, _ := autoSync(t, os, 10)
	require.NotNil(t, p.ExtractionSynchronizedAt)
	require.Equal(t, domain.ObserverReturning, os[0].Status)
}
func TestResolveExtractionSynchronization_LateStartUsesSemanticSyncTime(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, os, testutil.BaseTime.Add(20*time.Second), testutil.NewFakeRandom().QueueInt(10), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.NotNil(t, os[0].PhaseStartedAt)
	require.Equal(t, testutil.BaseTime.Add(cfg.ExtractionSync), *os[0].PhaseStartedAt)
}
func TestResolveExtractionSynchronization_OtherWaitingObserversRemainUnchanged(t *testing.T) {
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	before := os[1]
	autoSync(t, os, 10)
	require.Equal(t, before, os[1])
}
func TestResolveExtractionSynchronization_InvalidObserverStateIsAtomic(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	before := p
	plane := stage4Plane()
	bad := domain.Observer{ID: 1, Status: domain.ObserverWaitingReturn}
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, []domain.Observer{bad}, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrObserverInvariant)
	require.Equal(t, before, p)
}
func TestResolveExtractionSynchronization_BusyPortalIsAtomic(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	start := testutil.BaseTime
	end := start.Add(20 * time.Second)
	id := p.ID
	busy := domain.Observer{ID: 1, Status: domain.ObserverReturning, CurrentPlaneID: &plane.ID, ActivePortalID: &id, PhaseStartedAt: &start, PhaseEndsAt: &end}
	waiting := waitingObserverInPlane(2, plane.ID, start)
	before := p
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, []domain.Observer{busy, waiting}, start.Add(cfg.ExtractionSync), testutil.NewFakeRandom().QueueInt(10), cfg)
	require.ErrorIs(t, err, domain.ErrPortalBusy)
	require.Equal(t, before, p)
}
func TestResolveExtractionSynchronization_NilRandomWithWaitingObserverIsAtomic(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	before := p
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, os, testutil.BaseTime.Add(cfg.ExtractionSync), nil, cfg)
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
	require.Equal(t, before, p)
	require.Equal(t, domain.ObserverWaitingReturn, os[0].Status)
}
