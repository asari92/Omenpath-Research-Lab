package domain_test

// Final Spec §5: Portal Slots as pure domain logic.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func openAt(slot int) domain.Portal {
	return testutil.NewPortalBuilder().Slot(slot).Build()
}

// SLOT-002/003: with no OPEN portals the first slot is free.
func TestFirstFreeSlot_EmptyReturnsFirstSlot(t *testing.T) {
	slot, ok := domain.FirstFreeSlot(nil, config.Default().MaxActivePortals)
	require.True(t, ok)
	require.Equal(t, 1, slot)
}

// SLOT-003: the first free slot after the occupied prefix.
func TestFirstFreeSlot_ReturnsNextAfterOccupiedPrefix(t *testing.T) {
	portals := []domain.Portal{openAt(1), openAt(2)}
	slot, ok := domain.FirstFreeSlot(portals, config.Default().MaxActivePortals)
	require.True(t, ok)
	require.Equal(t, 3, slot)
}

// SLOT-003: gaps are filled first — slot 3 is taken before 4.
func TestFirstFreeSlot_FillsGap(t *testing.T) {
	portals := []domain.Portal{openAt(1), openAt(2), openAt(4)}
	slot, ok := domain.FirstFreeSlot(portals, config.Default().MaxActivePortals)
	require.True(t, ok)
	require.Equal(t, 3, slot)
}

// SLOT-005: terminal portals do not occupy slots.
func TestFirstFreeSlot_TerminalPortalsDoNotOccupy(t *testing.T) {
	terminal := testutil.NewPortalBuilder().Slot(1).TTL(10 * time.Second).Build()
	clk := testutil.NewFakeClock(testutil.BaseTime)
	clk.Advance(10 * time.Second)
	_, err := terminal.ResolveLifecycle(clk.Now())
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, terminal.Status)

	portals := []domain.Portal{terminal, openAt(2)}
	slot, ok := domain.FirstFreeSlot(portals, config.Default().MaxActivePortals)
	require.True(t, ok)
	require.Equal(t, 1, slot)
}

// SLOT-001/006: with all seven slots OPEN there is no free slot.
func TestFirstFreeSlot_ReturnsNoneWhenAllSevenOpen(t *testing.T) {
	portals := make([]domain.Portal, 0, 7)
	for s := 1; s <= 7; s++ {
		portals = append(portals, openAt(s))
	}

	slot, ok := domain.FirstFreeSlot(portals, config.Default().MaxActivePortals)
	require.False(t, ok, "7/7 OPEN portals must leave no free slot")
	require.Equal(t, 0, slot)
}

// SLOT-004: the helper is pure — it must not mutate or reorder the input.
func TestFirstFreeSlot_DoesNotMutateInput(t *testing.T) {
	portals := []domain.Portal{openAt(2), openAt(3)}
	snapshot := append([]domain.Portal(nil), portals...)

	_, ok := domain.FirstFreeSlot(portals, config.Default().MaxActivePortals)
	require.True(t, ok)
	require.Equal(t, snapshot, portals)
}
