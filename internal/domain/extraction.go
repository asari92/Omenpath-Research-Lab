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
