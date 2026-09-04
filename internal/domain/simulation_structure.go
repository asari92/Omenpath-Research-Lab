package domain

import (
	"math"
	"time"
)

// ValidateSimulationStructure checks config-independent aggregate invariants.
// It intentionally does not enforce tunable balance values such as extraction
// synchronization duration, creature maximums, or energy ranges.
func ValidateSimulationStructure(state SimulationState, now time.Time) error {
	return validateSimulationStructure(&state, now)
}

func validateSimulationStructure(state *SimulationState, now time.Time) error {
	if state == nil || len(state.Planes) != 85 ||
		len(state.Observers) == 0 || state.NextPortalID <= 0 ||
		state.Lab.EnergyBase < 0 || state.Lab.EnergyBaseAt.After(now) ||
		(state.LastTickAt != nil && state.LastTickAt.After(now)) ||
		!validNaturalSpawnState(state.NaturalSpawn, now) {
		return ErrSimulationInvariant
	}

	planeIDs := make(map[int64]struct{}, len(state.Planes))
	for _, plane := range state.Planes {
		if plane.ID <= 0 || plane.Explored != (plane.ExploredAt != nil) ||
			(plane.ExploredAt != nil && plane.ExploredAt.After(now)) {
			return ErrSimulationInvariant
		}
		if _, duplicate := planeIDs[plane.ID]; duplicate {
			return ErrSimulationInvariant
		}
		planeIDs[plane.ID] = struct{}{}
	}

	portalIDs := make(map[int64]int, len(state.Portals))
	openSlots := make(map[int]struct{}, 7)
	maxPortalID := int64(0)
	for i := range state.Portals {
		portal := &state.Portals[i]
		if !validStructuralPortal(portal, now) {
			return ErrSimulationInvariant
		}
		if _, duplicate := portalIDs[portal.ID]; duplicate {
			return ErrSimulationInvariant
		}
		if _, exists := planeIDs[portal.DestinationPlaneID]; !exists {
			return ErrSimulationInvariant
		}
		portalIDs[portal.ID] = i
		if portal.ID > maxPortalID {
			maxPortalID = portal.ID
		}
		if portal.Status == PortalStatusOpen {
			if _, duplicate := openSlots[portal.SlotIndex]; duplicate {
				return ErrSimulationInvariant
			}
			openSlots[portal.SlotIndex] = struct{}{}
		}
	}
	if state.NextPortalID <= maxPortalID {
		return ErrSimulationInvariant
	}
	return validateSimulationObservers(state.Observers, planeIDs, portalIDs, state.Portals, now)
}

func validStructuralPortal(portal *Portal, now time.Time) bool {
	if portal == nil || portal.ID <= 0 || portal.SlotIndex < 1 || portal.SlotIndex > 7 ||
		portal.DestinationPlaneID <= 0 || portal.OpenedAt.IsZero() || portal.CreatedAt.IsZero() ||
		portal.EnergyBaseAt.IsZero() || portal.UpdatedAt.IsZero() || portal.OpenedAt.After(now) ||
		!portal.ScheduledCloseAt.After(portal.OpenedAt) || portal.CreatedAt.After(portal.OpenedAt) ||
		portal.EnergyBaseAt.Before(portal.OpenedAt) || portal.EnergyBaseAt.After(portal.UpdatedAt) ||
		portal.EnergyBaseAt.After(now) || portal.UpdatedAt.Before(portal.OpenedAt) ||
		portal.UpdatedAt.Before(portal.CreatedAt) || portal.UpdatedAt.After(now) ||
		math.IsNaN(portal.EnergyBase) || math.IsInf(portal.EnergyBase, 0) ||
		math.IsNaN(portal.EnergyDecayRate) || math.IsInf(portal.EnergyDecayRate, 0) ||
		portal.EnergyBase < 0 || portal.EnergyDecayRate <= 0 || portal.CreaturesInitial < 0 {
		return false
	}
	if portal.ObserverFlow != PortalFlowNone && portal.ObserverFlow != PortalFlowOutbound &&
		portal.ObserverFlow != PortalFlowInbound {
		return false
	}

	switch portal.Stability {
	case PortalStable:
		if portal.InstabilityCollapseAt != nil {
			return false
		}
	case PortalUnstable:
		if portal.InstabilityCollapseAt == nil ||
			!portal.InstabilityCollapseAt.After(portal.OpenedAt) ||
			!portal.InstabilityCollapseAt.Before(portal.ScheduledCloseAt) {
			return false
		}
	default:
		return false
	}

	switch portal.Kind {
	case PortalKindNatural:
		if portal.ExtractionSynchronizedAt != nil {
			return false
		}
	case PortalKindExtraction:
		if portal.Stability != PortalStable || portal.ObserverFlow != PortalFlowInbound ||
			portal.CreaturesInitial != 0 {
			return false
		}
		if portal.ExtractionSynchronizedAt != nil &&
			(!portal.ExtractionSynchronizedAt.After(portal.OpenedAt) ||
				portal.ExtractionSynchronizedAt.After(now)) {
			return false
		}
	default:
		return false
	}

	switch portal.Status {
	case PortalStatusOpen:
		return portal.ClosedAt == nil && portal.TerminationReason == TerminationNone
	case PortalStatusClosed, PortalStatusCollapsed:
		return validSimulationTerminalPortal(portal, now)
	default:
		return false
	}
}
