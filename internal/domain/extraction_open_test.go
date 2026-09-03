package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func extractionFactoryRandom(cfg config.Config) *testutil.FakeRandom {
	return testutil.NewFakeRandom().QueueInt(int(cfg.ExtractionTTLMin.Seconds())).QueueFloat(cfg.ExtractionEnergyMin, cfg.PortalDecayMin)
}

func extractionOpenState(energy int) (domain.LabState, []domain.Portal, domain.Plane, []domain.Observer) {
	plane := stage4Plane()
	return labAt(energy, testutil.BaseTime), nil, plane,
		[]domain.Observer{waitingObserverInPlane(1, plane.ID, testutil.BaseTime.Add(-time.Minute))}
}

func openExtraction(t *testing.T, lab *domain.LabState, portals *[]domain.Portal, plane *domain.Plane, observers []domain.Observer, now time.Time, cfg config.Config) int64 {
	t.Helper()
	id, err := domain.OpenExtractionPortal(lab, portals, plane, observers, 42, now, extractionFactoryRandom(cfg), cfg)
	require.NoError(t, err)
	return id
}

func TestOpenExtractionPortal_SucceedsForSelectedEligiblePlane(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Len(t, ps, 1)
	require.Equal(t, p.ID, ps[0].DestinationPlaneID)
}
func TestOpenExtractionPortal_ReturnsCreatedPortalID(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	require.Equal(t, int64(42), openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default()))
}
func TestOpenExtractionPortal_AppendsExactlyOnePortal(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Len(t, ps, 1)
}
func TestOpenExtractionPortal_PreservesExistingPortalOrder(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	first := stage4Portal()
	first.ID = 1
	first.Status = domain.PortalStatusClosed
	ps := []domain.Portal{first}
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, int64(1), ps[0].ID)
	require.Equal(t, int64(42), ps[1].ID)
}
func TestOpenExtractionPortal_UsesFirstFreeRegularSlot(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	a, b := stage4Portal(), stage4Portal()
	a.ID, a.SlotIndex = 1, 1
	b.ID, b.SlotIndex = 2, 3
	ps := []domain.Portal{a, b}
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, 2, ps[2].SlotIndex)
}
func TestOpenExtractionPortal_ReusesTerminalPortalSlot(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	old := stage4Portal()
	old.ID, old.SlotIndex, old.Status = 1, 1, domain.PortalStatusClosed
	ps := []domain.Portal{old}
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, 1, ps[1].SlotIndex)
}
func TestOpenExtractionPortal_RejectsAllSevenSlotsOccupied(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	ps := make([]domain.Portal, 7)
	for i := range ps {
		ps[i] = stage4Portal()
		ps[i].ID = int64(i + 1)
		ps[i].SlotIndex = i + 1
	}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrNoFreePortalSlot)
}
func TestOpenExtractionPortal_RejectsNoWaitingObserverInSelectedPlane(t *testing.T) {
	l, ps, p, _ := extractionOpenState(100)
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, nil, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrNoWaitingObserver)
}
func TestOpenExtractionPortal_RejectsWaitingObserverOnlyInOtherPlane(t *testing.T) {
	l, ps, p, _ := extractionOpenState(100)
	os := []domain.Observer{waitingObserverInPlane(1, 8, testutil.BaseTime)}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrNoWaitingObserver)
}
func TestOpenExtractionPortal_ChargesThirty(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, 70, l.EnergyBase)
}
func TestOpenExtractionPortal_AllowsExactThirty(t *testing.T) {
	l, ps, p, os := extractionOpenState(30)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Zero(t, l.EnergyBase)
}
func TestOpenExtractionPortal_UsesRegeneratedEnergy(t *testing.T) {
	cfg := config.Default()
	l, ps, p, os := extractionOpenState(20)
	now := testutil.BaseTime.Add(10 * time.Second)
	openExtraction(t, &l, &ps, &p, os, now, cfg)
	require.Zero(t, l.EnergyBase)
	require.Equal(t, now, l.EnergyBaseAt)
}
func TestOpenExtractionPortal_ChargesThirtyDuringOverride(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = 30
	l := overrideLab(0, testutil.BaseTime, cfg)
	deadline := *l.LeylineOverrideUntil
	_, ps, p, os := extractionOpenState(100)
	now := testutil.BaseTime.Add(time.Second)
	openExtraction(t, &l, &ps, &p, os, now, cfg)
	require.Zero(t, l.EnergyBase)
	require.Equal(t, deadline, *l.LeylineOverrideUntil)
}
func TestOpenExtractionPortal_PreservesOverrideDeadline(t *testing.T) {
	TestOpenExtractionPortal_ChargesThirtyDuringOverride(t)
}

func rejectedOpen(t *testing.T, mutate func(*domain.LabState, *[]domain.Portal, *domain.Plane, *[]domain.Observer, *config.Config), want error) {
	t.Helper()
	cfg := config.Default()
	l, ps, p, os := extractionOpenState(29)
	mutate(&l, &ps, &p, &os, &cfg)
	lb, pb, ob := l, append([]domain.Portal(nil), ps...), append([]domain.Observer(nil), os...)
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, want)
	require.Equal(t, lb, l)
	require.Equal(t, pb, ps)
	require.Equal(t, ob, os)
}
func TestOpenExtractionPortal_RejectsTwentyNine(t *testing.T) {
	rejectedOpen(t, func(*domain.LabState, *[]domain.Portal, *domain.Plane, *[]domain.Observer, *config.Config) {}, domain.ErrInsufficientLabEnergy)
}
func TestOpenExtractionPortal_InsufficientEnergyLeavesLabUnchanged(t *testing.T) {
	TestOpenExtractionPortal_RejectsTwentyNine(t)
}
func TestOpenExtractionPortal_InsufficientEnergyLeavesPortalsUnchanged(t *testing.T) {
	TestOpenExtractionPortal_RejectsTwentyNine(t)
}
func TestOpenExtractionPortal_InsufficientEnergyLeavesObserversUnchanged(t *testing.T) {
	TestOpenExtractionPortal_RejectsTwentyNine(t)
}
func TestOpenExtractionPortal_RejectionDoesNotConsumeRandom(t *testing.T) {
	TestOpenExtractionPortal_RejectsNoWaitingObserverInSelectedPlane(t)
}
func TestOpenExtractionPortal_InvalidLabIsAtomic(t *testing.T) {
	rejectedOpen(t, func(l *domain.LabState, _ *[]domain.Portal, _ *domain.Plane, _ *[]domain.Observer, _ *config.Config) {
		l.EnergyBase = -1
	}, domain.ErrLabEnergyInvariant)
}
func TestOpenExtractionPortal_NilPlaneIsAtomic(t *testing.T) {
	l, ps, _, os := extractionOpenState(100)
	_, err := domain.OpenExtractionPortal(&l, &ps, nil, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_NilPortalCollectionIsAtomic(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	_, err := domain.OpenExtractionPortal(&l, nil, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_RejectsDuplicatePortalID(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	old := stage4Portal()
	old.ID = 42
	ps := []domain.Portal{old}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_RejectsDuplicateOpenSlot(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	a, b := stage4Portal(), stage4Portal()
	a.ID = 1
	b.ID = 2
	a.SlotIndex = 1
	b.SlotIndex = 1
	ps := []domain.Portal{a, b}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_RejectsOutOfRangeOpenSlot(t *testing.T) {
	l, _, p, os := extractionOpenState(100)
	a := stage4Portal()
	a.ID = 1
	a.SlotIndex = 8
	ps := []domain.Portal{a}
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_InvalidConfigIsAtomic(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	cfg := config.Default()
	cfg.ExtractionSync = 0
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_NilRandomIsAtomic(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	_, err := domain.OpenExtractionPortal(&l, &ps, &p, os, 42, testutil.BaseTime, nil, config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestOpenExtractionPortal_DoesNotSelectOrReserveObserver(t *testing.T) {
	l, ps, p, os := extractionOpenState(100)
	before := append([]domain.Observer(nil), os...)
	openExtraction(t, &l, &ps, &p, os, testutil.BaseTime, config.Default())
	require.Equal(t, before, os)
}
