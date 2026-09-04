package domain

import (
	"fmt"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// NewExtractionPortal creates the controlled INBOUND portal from Final Spec
// §20. Plane selection, slot allocation and Laboratory Energy are orchestrated
// by OpenExtractionPortal in later checkpoints.
func NewExtractionPortal(seq, planeID int64, slot int, now time.Time, cfg config.Config, rnd random.Random) Portal {
	ttl := time.Duration(rnd.IntInclusive(
		int(cfg.ExtractionTTLMin.Seconds()),
		int(cfg.ExtractionTTLMax.Seconds()),
	)) * time.Second
	energy := rnd.FloatRange(cfg.ExtractionEnergyMin, cfg.ExtractionEnergyMax)
	decay := rnd.FloatRange(cfg.PortalDecayMin, cfg.PortalDecayMax)

	return Portal{
		ID:                 seq,
		Name:               fmt.Sprintf("Omenpath #%04d", seq),
		SlotIndex:          slot,
		Kind:               PortalKindExtraction,
		DestinationPlaneID: planeID,

		EnergyBase:      energy,
		EnergyBaseAt:    now,
		EnergyDecayRate: decay,

		Stability: PortalStable,

		OpenedAt:         now,
		ScheduledCloseAt: now.Add(ttl),

		ObserverFlow: PortalFlowInbound,

		Status:            PortalStatusOpen,
		TerminationReason: TerminationNone,

		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ExtractionPlaneEligible reports whether planeID currently contains at
// least one canonical WAITING_RETURN Observer.
func ExtractionPlaneEligible(observers []Observer, planeID int64, now time.Time) (bool, error) {
	if err := validateObserverRoster(observers, now); err != nil {
		return false, err
	}
	_, ok, err := LongestWaitingObserverIndex(observers, planeID)
	return ok, err
}

// OpenExtractionPortal atomically charges Laboratory Energy and appends one
// controlled portal in the first regular free slot.
func OpenExtractionPortal(
	lab *LabState,
	portals *[]Portal,
	plane *Plane,
	observers []Observer,
	seq int64,
	now time.Time,
	rnd random.Random,
	cfg config.Config,
) (int64, error) {
	if _, err := prepareLabEnergySpend(lab, now, 0, cfg); err != nil {
		return 0, err
	}
	if portals == nil || plane == nil || rnd == nil || !validExtractionConfig(cfg) {
		return 0, ErrExtractionInvariant
	}
	eligible, err := ExtractionPlaneEligible(observers, plane.ID, now)
	if err != nil {
		return 0, err
	}
	if !eligible {
		return 0, ErrNoWaitingObserver
	}
	if err := validateExtractionPortalCollection(*portals, seq, cfg.MaxActivePortals); err != nil {
		return 0, err
	}
	slot, ok := FirstFreeSlot(*portals, cfg.MaxActivePortals)
	if !ok {
		return 0, ErrNoFreePortalSlot
	}
	if _, err := prepareLabEnergySpend(lab, now, cfg.ExtractionCost, cfg); err != nil {
		return 0, err
	}

	nextLab := *lab
	if err := nextLab.SpendEnergy(now, cfg.ExtractionCost, cfg); err != nil {
		return 0, err
	}
	portal := NewExtractionPortal(seq, plane.ID, slot, now, cfg, rnd)
	nextPortals := append(append([]Portal(nil), (*portals)...), portal)
	*lab = nextLab
	*portals = nextPortals
	return portal.ID, nil
}

func validExtractionConfig(cfg config.Config) bool {
	return cfg.MaxActivePortals > 0 && cfg.ExtractionCost >= 0 &&
		cfg.ExtractionEnergyMin >= 0 && cfg.ExtractionEnergyMin <= cfg.ExtractionEnergyMax &&
		cfg.PortalDecayMin > 0 && cfg.PortalDecayMin <= cfg.PortalDecayMax &&
		cfg.ExtractionTTLMin > 0 && cfg.ExtractionTTLMin <= cfg.ExtractionTTLMax &&
		cfg.ExtractionSync > 0
}

func validateExtractionPortalCollection(portals []Portal, newID int64, maxSlots int) error {
	ids := make(map[int64]struct{}, len(portals))
	openSlots := make(map[int]struct{}, maxSlots)
	for _, portal := range portals {
		if portal.ID == newID {
			return ErrExtractionInvariant
		}
		if _, duplicate := ids[portal.ID]; duplicate {
			return ErrExtractionInvariant
		}
		ids[portal.ID] = struct{}{}
		if portal.Status != PortalStatusOpen {
			continue
		}
		if portal.SlotIndex < 1 || portal.SlotIndex > maxSlots {
			return ErrExtractionInvariant
		}
		if _, duplicate := openSlots[portal.SlotIndex]; duplicate {
			return ErrExtractionInvariant
		}
		openSlots[portal.SlotIndex] = struct{}{}
	}
	return nil
}
