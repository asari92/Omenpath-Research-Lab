package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func newExtractionFixture(ttl int, energy, decay float64) domain.Portal {
	cfg := config.Default()
	rnd := testutil.NewFakeRandom().QueueInt(ttl).QueueFloat(energy, decay)
	return domain.NewExtractionPortal(42, 7, 3, testutil.BaseTime, cfg, rnd)
}

func TestNewExtractionPortal_SetsExtractionKind(t *testing.T) {
	require.Equal(t, domain.PortalKindExtraction, newExtractionFixture(30, 60, .1).Kind)
}

func TestNewExtractionPortal_StartsOpen(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, domain.PortalStatusOpen, p.Status)
	require.Equal(t, domain.TerminationNone, p.TerminationReason)
}

func TestNewExtractionPortal_IsStableWithoutHiddenCollapse(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, domain.PortalStable, p.Stability)
	require.Nil(t, p.InstabilityCollapseAt)
}

func TestNewExtractionPortal_StartsInbound(t *testing.T) {
	require.Equal(t, domain.PortalFlowInbound, newExtractionFixture(30, 60, .1).ObserverFlow)
}

func TestNewExtractionPortal_HasNoCreatures(t *testing.T) {
	require.Zero(t, newExtractionFixture(30, 60, .1).CreaturesInitial)
}

func TestNewExtractionPortal_UsesSelectedPlaneAndSlot(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, int64(7), p.DestinationPlaneID)
	require.Equal(t, 3, p.SlotIndex)
}

func TestNewExtractionPortal_UsesSequentialIdentity(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, int64(42), p.ID)
	require.Equal(t, "Omenpath #0042", p.Name)
}

func TestNewExtractionPortal_AcceptsMinimumEnergy(t *testing.T) {
	require.Equal(t, 60.0, newExtractionFixture(30, 60, .1).EnergyBase)
}

func TestNewExtractionPortal_AcceptsMaximumEnergy(t *testing.T) {
	require.Equal(t, 100.0, newExtractionFixture(30, 100, .1).EnergyBase)
}

func TestNewExtractionPortal_AcceptsMinimumDecay(t *testing.T) {
	require.Equal(t, .1, newExtractionFixture(30, 60, .1).EnergyDecayRate)
}

func TestNewExtractionPortal_AcceptsMaximumDecay(t *testing.T) {
	require.Equal(t, 1.0, newExtractionFixture(30, 60, 1).EnergyDecayRate)
}

func TestNewExtractionPortal_AcceptsMinimumTTL(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, testutil.BaseTime.Add(30*time.Second), p.ScheduledCloseAt)
}

func TestNewExtractionPortal_AcceptsMaximumTTL(t *testing.T) {
	p := newExtractionFixture(60, 60, .1)
	require.Equal(t, testutil.BaseTime.Add(60*time.Second), p.ScheduledCloseAt)
}

type extractionRecordingRandom struct {
	calls  []string
	floats []float64
}

func (r *extractionRecordingRandom) IntInclusive(_, _ int) int {
	r.calls = append(r.calls, "int")
	return 30
}

func (r *extractionRecordingRandom) FloatRange(_, _ float64) float64 {
	r.calls = append(r.calls, "float")
	v := r.floats[0]
	r.floats = r.floats[1:]
	return v
}

func TestNewExtractionPortal_DrawsTTLThenEnergyThenDecay(t *testing.T) {
	rnd := &extractionRecordingRandom{floats: []float64{60, .1}}
	domain.NewExtractionPortal(42, 7, 3, testutil.BaseTime, config.Default(), rnd)
	require.Equal(t, []string{"int", "float", "float"}, rnd.calls)
}

func TestNewExtractionPortal_InitializesTimestamps(t *testing.T) {
	p := newExtractionFixture(30, 60, .1)
	require.Equal(t, testutil.BaseTime, p.OpenedAt)
	require.Equal(t, testutil.BaseTime, p.EnergyBaseAt)
	require.Equal(t, testutil.BaseTime, p.CreatedAt)
	require.Equal(t, testutil.BaseTime, p.UpdatedAt)
	require.Nil(t, p.ClosedAt)
}

func TestNewExtractionPortal_StartsUnsynchronized(t *testing.T) {
	require.Nil(t, newExtractionFixture(30, 60, .1).ExtractionSynchronizedAt)
}

func TestNewExtractionPortal_DoesNotMutateUnrelatedState(t *testing.T) {
	cfg := config.Default()
	before := cfg
	domain.NewExtractionPortal(42, 7, 3, testutil.BaseTime, cfg,
		testutil.NewFakeRandom().QueueInt(30).QueueFloat(60, .1))
	require.Equal(t, before, cfg)
}
