package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestExtraction_OpenSyncReturnEndToEnd(t *testing.T) {
	cfg := config.Default()
	lab, portals, plane, os := extractionOpenState(100)
	id := openExtraction(t, &lab, &portals, &plane, os, testutil.BaseTime, cfg)
	require.Equal(t, int64(42), id)
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	returned, changed, err := domain.ResolveExtractionSynchronization(&portals[0], &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, int64(1), returned)
	require.NoError(t, domain.ResolveObserverLifecycle(&os[0], &plane, &portals[0], syncAt.Add(5*time.Second), cfg))
	require.Equal(t, domain.ObserverAvailable, os[0].Status)
	require.Equal(t, 70, lab.EnergyBase)
}
func TestExtraction_OpenDoesNotMutatePlane(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	before := p
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, before, p)
}
func TestExtraction_OpenDoesNotMutateObserverRoster(t *testing.T) {
	TestOpenExtractionPortal_DoesNotSelectOrReserveObserver(t)
}
func TestExtraction_OpenAndPortalEnergyRemainIndependent(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, 70, l.EnergyBase)
	require.Equal(t, 60.0, ps[0].EnergyBase)
}
func TestExtraction_OpeningTwoPortalsCreatesDistinctInstances(t *testing.T) {
	cfg := config.Default()
	l, ps, p, os := extractionOpenState(100)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, cfg)
	id, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 43, testutil.BaseTime, extractionFactoryRandom(cfg), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(43), id)
	require.Len(t, ps, 2)
	require.NotEqual(t, ps[0].ID, ps[1].ID)
	require.NotEqual(t, ps[0].SlotIndex, ps[1].SlotIndex)
}
func TestExtraction_TwoPortalsSynchronizeAtMostOnceEach(t *testing.T) {
	cfg := config.Default()
	p1, p2 := extractionPortal(testutil.BaseTime, cfg), extractionPortal(testutil.BaseTime, cfg)
	p2.ID = 43
	p2.SlotIndex = 2
	plane := stage4Plane()
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime.Add(-time.Minute)), waitingObserverInPlane(2, 7, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	id1, ch1, e1 := domain.ResolveExtractionSynchronization(&p1, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	id2, ch2, e2 := domain.ResolveExtractionSynchronization(&p2, &plane, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, e1)
	require.NoError(t, e2)
	require.True(t, ch1)
	require.True(t, ch2)
	require.Equal(t, int64(1), id1)
	require.Equal(t, int64(2), id2)
	_, ch1, e1 = domain.ResolveExtractionSynchronization(&p1, &plane, os, syncAt.Add(time.Second), testutil.NewFakeRandom(), cfg)
	require.NoError(t, e1)
	require.False(t, ch1)
}
func TestExtraction_DifferentPlanesSelectIndependentObservers(t *testing.T) {
	cfg := config.Default()
	p1 := extractionPortal(testutil.BaseTime, cfg)
	p2 := extractionPortal(testutil.BaseTime, cfg)
	p2.ID = 43
	p2.DestinationPlaneID = 8
	plane1 := stage4Plane()
	plane2 := domain.Plane{ID: 8}
	os := []domain.Observer{waitingObserverInPlane(1, 7, testutil.BaseTime), waitingObserverInPlane(2, 8, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	id1, _, e1 := domain.ResolveExtractionSynchronization(&p1, &plane1, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	id2, _, e2 := domain.ResolveExtractionSynchronization(&p2, &plane2, os, syncAt, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, e1)
	require.NoError(t, e2)
	require.Equal(t, int64(1), id1)
	require.Equal(t, int64(2), id2)
}
func TestExtraction_NoOpeningTimeObserverReservation(t *testing.T) {
	TestOpenExtractionPortal_DoesNotSelectOrReserveObserver(t)
}
func TestExtraction_NoWaitingAtSyncKeepsEverythingElseOperational(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Equal(t, domain.PortalStatusOpen, p.Status)
	require.Equal(t, domain.PortalFlowInbound, p.ObserverFlow)
}
func TestExtraction_NoWaitingAtSyncAllowsLaterManualRecall(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, syncAt, testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	os := []domain.Observer{waitingObserverInPlane(1, 7, syncAt)}
	id, err := domain.RecallObserver(&p, &plane, os, syncAt.Add(time.Second), false, testutil.NewFakeRandom().QueueInt(5), cfg)
	require.NoError(t, err)
	require.Equal(t, int64(1), id)
}
func TestExtraction_InvalidOpenAggregateIsAtomic(t *testing.T) {
	TestOpenExtractionPortal_RejectsDuplicateOpenSlot(t)
}
func TestExtraction_InvalidSyncAggregateIsAtomic(t *testing.T) {
	TestResolveExtractionSynchronization_InvalidObserverStateIsAtomic(t)
}
func TestExtraction_ErrorIdentitySurvivesEnergyWrapper(t *testing.T) {
	TestRecallObserverWithLabEnergy_InheritsExtractionSyncGate(t)
}
func TestExtraction_NaturalFactoryStillStartsFlowNone(t *testing.T) {
	p := testutil.NewPortalBuilder().Build()
	require.Equal(t, domain.PortalFlowNone, p.ObserverFlow)
	require.Nil(t, p.ExtractionSynchronizedAt)
}
func TestExtraction_NaturalRecallStillWorksBeforeFiveSeconds(t *testing.T) {
	TestRecallObserver_NaturalPortalBehaviorUnchangedByExtractionGate(t)
}
func TestExtraction_OverrideWindowSurvivesOpeningDebit(t *testing.T) {
	TestOpenExtractionPortal_ChargesThirtyDuringOverride(t)
}
func TestExtraction_OpeningRejectionPrecedenceIsDeterministic(t *testing.T) {
	cfg := config.Default()
	l := labAt(0, testutil.BaseTime)
	p := stage4Plane()
	ps := make([]domain.Portal, 7)
	for i := range ps {
		ps[i] = stage4Portal()
		ps[i].ID = int64(i + 1)
		ps[i].SlotIndex = i + 1
	}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, nil, 42, testutil.BaseTime, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrNoWaitingObserver)
}

func TestExtraction_LateSynchronizationReplayIsIdempotent(t *testing.T) {
	cfg := config.Default()
	portal := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserverInPlane(1, plane.ID, testutil.BaseTime)}
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)

	_, changed, err := domain.ResolveExtractionSynchronization(
		&portal,
		&plane,
		observers,
		syncAt,
		testutil.NewFakeRandom().QueueInt(5),
		cfg,
	)
	require.NoError(t, err)
	require.True(t, changed)
	require.NotNil(t, observers[0].PhaseEndsAt)

	_, changed, err = domain.ResolveExtractionSynchronization(
		&portal,
		&plane,
		observers,
		observers[0].PhaseEndsAt.Add(time.Second),
		testutil.NewFakeRandom(),
		cfg,
	)
	require.NoError(t, err)
	require.False(t, changed)
}
