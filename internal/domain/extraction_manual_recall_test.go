package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func markExtractionSynced(p *domain.Portal, cfg config.Config) {
	at := p.OpenedAt.Add(cfg.ExtractionSync)
	p.ExtractionSynchronizedAt = &at
}

func TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	before := os[1]
	_, changed, err = domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt.Add(time.Second), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, before, os[1])
}
func TestResolveExtractionSynchronization_ReplayDoesNotConsumeRandom(t *testing.T) {
	TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver(t)
}
func TestResolveExtractionSynchronization_ObserverAddedLaterIsNotAutomatic(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, syncAt, testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	os := []domain.Observer{waitingObserverInPlane(1, 7, syncAt)}
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt.Add(time.Second), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, domain.ObserverWaitingReturn, os[0].Status)
}
func TestResolveExtractionSynchronization_MalformedMarkerRejectsAtomically(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	bad := testutil.BaseTime.Add(time.Second)
	p.ExtractionSynchronizedAt = &bad
	before := p
	plane := stage4Plane()
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
	require.Equal(t, before, p)
}
func TestRecallObserver_ExtractionBeforeSyncRejects(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, &extractionRecordingRandom{}, cfg)
	require.ErrorIs(t, err, domain.ErrExtractionSynchronizing)
}
func TestRecallObserver_ExtractionBeforeSyncRejectionIsAtomic(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	before := p
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	obefore := os[0]
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, &extractionRecordingRandom{}, cfg)
	require.Error(t, err)
	require.Equal(t, before, p)
	require.Equal(t, obefore, os[0])
}
func TestRecallObserver_ExtractionBeforeSyncConsumesNoRandom(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	rnd := &extractionRecordingRandom{}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, rnd, cfg)
	require.ErrorIs(t, err, domain.ErrExtractionSynchronizing)
	require.Empty(t, rnd.calls)
}
func TestRecallObserver_ExtractionAtDeadlineBeforeResolverStillRejects(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime.Add(cfg.ExtractionSync), false, &extractionRecordingRandom{}, cfg)
	require.ErrorIs(t, err, domain.ErrExtractionSynchronizing)
}
func TestRecallObserver_ExtractionAfterSyncAllowsManualReturn(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	markExtractionSynced(&p, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	id, err := domain.RecallObserver(&p, &plane, os, *p.ExtractionSynchronizedAt, false, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
}
func TestRecallObserver_ExtractionAfterSyncSelectsLongestWaiting(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	markExtractionSynced(&p, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(2, 7, testutil.BaseTime), waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute))}
	id, err := domain.RecallObserver(&p, &plane, os, *p.ExtractionSynchronizedAt, false, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
}
func TestRecallObserverWithLabEnergy_InheritsExtractionSyncGate(t *testing.T) {
	cfg := config.Default()
	lab := labAt(0, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserverWithLabEnergy(&lab, &p, &plane, os, testutil.BaseTime, false, &extractionRecordingRandom{}, cfg)
	require.ErrorIs(t, err, domain.ErrExtractionSynchronizing)
}
func TestRecallObserverWithLabEnergy_AfterSyncRemainsZeroCost(t *testing.T) {
	cfg := config.Default()
	lab := labAt(0, testutil.BaseTime)
	before := lab
	p := extractionPortal(testutil.BaseTime, cfg)
	markExtractionSynced(&p, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserverWithLabEnergy(&lab, &p, &plane, os, *p.ExtractionSynchronizedAt, false, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.Equal(t, before, lab)
}
func TestExtractionPortal_AutomaticTransitBlocksManualRecallWhileBusy(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(10), cfg)
	require.NoError(t, err)
	_, err = domain.RecallObserver(&p, &plane, os, syncAt, false, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrPortalBusy)
}
func TestExtractionPortal_AfterAutomaticReturnAllowsNextManualRecall(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.NoError(t, domain.ResolveObserverLifecycle(&os[0], &plane, &p, syncAt.Add(5*time.Second), cfg))
	id, err := domain.RecallObserver(&p, &plane, os, syncAt.Add(5*time.Second), false, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(2), id)
}
func TestExtractionPortal_OnlyFirstReturnIsAutomatic(t *testing.T) {
	TestResolveExtractionSynchronization_ReplayDoesNotStartSecondObserver(t)
}
func TestRecallObserver_NaturalPortalBehaviorUnchangedByExtractionGate(t *testing.T) {
	p := stage4Portal()
	p.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(5), config.Default())
	require.NoError(t, err)
}
func TestRecallObserver_ClosedUnsynchronizedExtractionReportsNotOpen(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	p.Status = domain.PortalStatusClosed
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	_, err := domain.RecallObserver(&p, &plane, os, testutil.BaseTime, false, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}
