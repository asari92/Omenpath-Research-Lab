package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func sendWithLab(t *testing.T, lab *domain.LabState, portal *domain.Portal, plane *domain.Plane, observers []domain.Observer, now time.Time, rnd *testutil.FakeRandom) int64 {
	t.Helper()
	id, err := domain.SendObserverWithLabEnergy(lab, portal, plane, observers, now, false, rnd, config.Default())
	require.NoError(t, err)
	return id
}

func TestSendObserverWithLabEnergy_AllowsZeroEnergy(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	id := sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, int64(1), id)
}

func TestSendObserverWithLabEnergy_CostsZero(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, 40, lab.EnergyBase)
}

func TestSendObserverWithLabEnergy_DoesNotRebaseEnergy(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime.Add(3*time.Second), testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, testutil.BaseTime, lab.EnergyBaseAt)
}

func TestSendObserverWithLabEnergy_DelegatesStage4Selection(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(8, testutil.BaseTime), domain.NewObserver(2, testutil.BaseTime)}

	id := sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, int64(2), id)
}

func TestSendObserverWithLabEnergy_DelegatesStage4Flow(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))
	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
}

func TestSendObserverWithLabEnergy_PreservesUnstableConfirmation(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
}

func TestSendObserverWithLabEnergy_RejectionDoesNotChangeLab(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, snapshot, lab)
}

func TestSendObserverWithLabEnergy_RejectionDoesNotConsumeRandom(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	rnd := testutil.NewFakeRandom().QueueInt(13)

	_, _ = domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}

func TestSendObserverWithLabEnergy_SuccessDrawsTransitOnce(t *testing.T) {
	lab := labAt(0, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	sendWithLab(t, &lab, &portal, &plane, observers, testutil.BaseTime, rnd)
	require.Equal(t, 14, rnd.IntInclusive(5, 15))
}

func TestSendObserverWithLabEnergy_RejectsInvalidLabStateBeforeMutation(t *testing.T) {
	lab := labAt(-1, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)
	rnd := testutil.NewFakeRandom().QueueInt(13)

	_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}
