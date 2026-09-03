package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func recallWithLab(t *testing.T, lab *domain.LabState, portal *domain.Portal, plane *domain.Plane, observers []domain.Observer, now time.Time, rnd *testutil.FakeRandom) int64 {
	t.Helper()
	id, err := domain.RecallObserverWithLabEnergy(lab, portal, plane, observers, now, false, rnd, config.Default())
	require.NoError(t, err)
	return id
}

func TestRecallObserverWithLabEnergy_AllowsZeroEnergy(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	id := recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, int64(1), id)
}

func TestRecallObserverWithLabEnergy_CostsZero(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, 40, lab.EnergyBase)
}

func TestRecallObserverWithLabEnergy_DoesNotRebaseEnergy(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime.Add(3*time.Second), testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, testutil.BaseTime, lab.EnergyBaseAt)
}

func TestRecallObserverWithLabEnergy_DelegatesLongestWaitingSelection(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	newer := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	newer.ID = 8
	older := waitingObserver(testutil.BaseTime.Add(-2 * time.Minute))
	older.ID = 2
	observers := []domain.Observer{newer, older}

	id := recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, int64(2), id)
}

func TestRecallObserverWithLabEnergy_DelegatesStage4Flow(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, domain.PortalFlowInbound, portal.ObserverFlow)
}

func TestRecallObserverWithLabEnergy_PreservesUnstableConfirmation(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
}

func TestRecallObserverWithLabEnergy_RejectionDoesNotChangeLab(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, snapshot, lab)
}

func TestRecallObserverWithLabEnergy_RejectionDoesNotConsumeRandom(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	rnd := testutil.NewFakeRandom().QueueInt(13)

	_, _ = domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}

func TestRecallObserverWithLabEnergy_SuccessDrawsTransitOnce(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	recallWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, rnd)
	require.Equal(t, 14, rnd.IntInclusive(5, 15))
}

func TestRecallObserverWithLabEnergy_RejectsInvalidLabStateBeforeMutation(t *testing.T) {
	lab := labAt(-1, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)
	rnd := testutil.NewFakeRandom().QueueInt(13)

	_, err := domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}
