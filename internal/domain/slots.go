package domain

// FirstFreeSlot returns the first free slot in 1..maxSlots considering
// only OPEN portals as occupants (Final Spec §5).
//
// It is a pure helper: it never mutates or reorders existing portals —
// a portal keeps its slot for its whole lifecycle (SLOT-004), and
// re-sorting is explicitly out of scope (Final Spec §5, Worklog).
func FirstFreeSlot(portals []Portal, maxSlots int) (slot int, ok bool) {
	occupied := make(map[int]struct{}, maxSlots)
	for _, p := range portals {
		if p.Status == PortalStatusOpen {
			occupied[p.SlotIndex] = struct{}{}
		}
	}
	for s := 1; s <= maxSlots; s++ {
		if _, taken := occupied[s]; !taken {
			return s, true
		}
	}
	return 0, false
}
