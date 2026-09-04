package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func syncedReturningExtraction(t *testing.T) (domain.Portal, domain.Plane, []domain.Observer, time.Time) {
	t.Helper()
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(10), cfg)
	require.NoError(t, err)
	return p, plane, os, syncAt
}

func TestExtractionClose_DuringSyncClosesPortal(t *testing.T) {
	cfg := config.Default()
	lab := labAt(100, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	err := domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, testutil.BaseTime.Add(time.Second), false, cfg)
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, p.Status)
}
func TestExtractionClose_DuringSyncLeavesWaitingObserverUnchanged(t *testing.T) {
	cfg := config.Default()
	lab := labAt(100, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	before := os[0]
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, testutil.BaseTime.Add(time.Second), false, cfg))
	require.Equal(t, before, os[0])
}
func TestExtractionClose_DuringSyncDoesNotSetSynchronizationMarker(t *testing.T) {
	cfg := config.Default()
	lab := labAt(100, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, nil, testutil.BaseTime.Add(time.Second), false, cfg))
	require.Nil(t, p.ExtractionSynchronizedAt)
}
func TestExtractionClose_ResolverAfterPreSyncCloseIsNoOp(t *testing.T) {
	cfg := config.Default()
	lab := labAt(100, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime)}
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, testutil.BaseTime.Add(time.Second), false, cfg))
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, os, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, domain.ObserverWaitingReturn, os[0].Status)
}
func TestExtractionClose_DuringReturningRequiresConfirmation(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	lab := labAt(100, testutil.BaseTime)
	err := domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, syncAt.Add(time.Second), false, config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
}
func TestExtractionClose_RejectedConfirmationIsAtomic(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	lab := labAt(100, testutil.BaseTime)
	pb, lb, ob := p, lab, os[0]
	err := domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, syncAt.Add(time.Second), false, config.Default())
	require.Error(t, err)
	require.Equal(t, pb, p)
	require.Equal(t, lb, lab)
	require.Equal(t, ob, os[0])
}
func TestExtractionClose_ConfirmedReturningBecomesLost(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	lab := labAt(100, testutil.BaseTime)
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, syncAt.Add(time.Second), true, config.Default()))
	require.Equal(t, domain.ObserverLost, os[0].Status)
}
func TestExtractionClose_ConfirmedSetsManualCloseReason(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	lab := labAt(100, testutil.BaseTime)
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, syncAt.Add(time.Second), true, config.Default()))
	require.Equal(t, domain.TerminationManualClose, p.TerminationReason)
}
func TestExtractionClose_ConfirmedUsesSameTimestampForPortalAndLoss(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	lab := labAt(100, testutil.BaseTime)
	now := syncAt.Add(time.Second)
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, now, true, config.Default()))
	require.Equal(t, now, *p.ClosedAt)
	require.Equal(t, now, os[0].UpdatedAt)
}
func TestExtractionLifecycle_CollapseDuringReturningBecomesLost(t *testing.T) {
	cfg := config.Default()
	p, plane, os, syncAt := syncedReturningExtraction(t)
	p.EnergyBase, p.EnergyDecayRate, p.EnergyBaseAt = 2, 1, syncAt
	lab := labAt(100, testutil.BaseTime)
	collapseAt := syncAt.Add(2 * time.Second)
	_, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &p, collapseAt, cfg)
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusCollapsed, p.Status)
	require.NoError(t, domain.ResolveObserverLifecycle(&os[0], &plane, &p, collapseAt, cfg))
	require.Equal(t, domain.ObserverLost, os[0].Status)
}
func TestExtractionLifecycle_OtherWaitingObserversRemainInPlane(t *testing.T) {
	p, plane, os, syncAt := syncedReturningExtraction(t)
	other := waitingObserverInPlane(2, 7, testutil.BaseTime)
	os = append(os, other)
	lab := labAt(100, testutil.BaseTime)
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, os, syncAt.Add(time.Second), true, config.Default()))
	require.Equal(t, other, os[1])
}
func TestExtractionLifecycle_CloseAtReturnDeadlineSucceedsReturn(t *testing.T) {
	p, plane, os, _ := syncedReturningExtraction(t)
	end := *os[0].PhaseEndsAt
	require.NoError(t, p.Close(end, false, config.Default()))
	require.NoError(t, domain.ResolveObserverLifecycle(&os[0], &plane, &p, end, config.Default()))
	require.Equal(t, domain.ObserverAvailable, os[0].Status)
}
func TestExtractionLifecycle_CloseAfterReturnDoesNotRetroactivelyLoseObserver(t *testing.T) {
	p, plane, os, _ := syncedReturningExtraction(t)
	end := *os[0].PhaseEndsAt
	require.NoError(t, domain.ResolveObserverLifecycle(&os[0], &plane, &p, end, config.Default()))
	require.NoError(t, p.Close(end.Add(time.Second), false, config.Default()))
	require.Equal(t, domain.ObserverAvailable, os[0].Status)
}
func TestExtractionClose_DuringSyncUsesOrdinaryCloseCost(t *testing.T) {
	cfg := config.Default()
	lab := labAt(100, testutil.BaseTime)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, nil, testutil.BaseTime.Add(time.Second), false, cfg))
	require.Equal(t, 95, lab.EnergyBase)
}
func TestExtractionClose_DuringOverrideUsesFreeCloseRule(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &p, &plane, nil, testutil.BaseTime.Add(time.Second), false, cfg))
	require.Zero(t, lab.EnergyBase)
}
