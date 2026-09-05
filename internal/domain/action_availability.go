package domain

import (
	"errors"
	"time"

	"omenpath-lab/internal/config"
)

type PortalAction string

const (
	PortalActionStabilize PortalAction = "STABILIZE"
	PortalActionClose     PortalAction = "CLOSE"
	PortalActionSend      PortalAction = "SEND"
	PortalActionRecall    PortalAction = "RECALL"
)

type ActionAvailability struct {
	Available bool
	Cause     error
}

// PortalActionAvailability runs the ordinary command validation against an
// isolated aggregate copy with confirmation granted. It never mutates the
// caller's state or consumes the manager's Random source.
func PortalActionAvailability(
	state SimulationState,
	portalID int64,
	action PortalAction,
	now time.Time,
	cfg config.Config,
) (ActionAvailability, error) {
	if err := ValidateSimulationStructure(state, now); err != nil {
		return ActionAvailability{}, err
	}
	working := cloneSimulationState(state)
	portalIndex := -1
	for i := range working.Portals {
		if working.Portals[i].ID == portalID {
			portalIndex = i
			break
		}
	}
	if portalIndex < 0 {
		return ActionAvailability{}, ErrSimulationInvariant
	}
	portal := &working.Portals[portalIndex]
	planeIndex := -1
	for i := range working.Planes {
		if working.Planes[i].ID == portal.DestinationPlaneID {
			planeIndex = i
			break
		}
	}
	if planeIndex < 0 {
		return ActionAvailability{}, ErrSimulationInvariant
	}

	var err error
	switch action {
	case PortalActionStabilize:
		err = StabilizePortalWithLabEnergy(&working.Lab, portal, now, cfg)
	case PortalActionClose:
		err = ClosePortalWithLabEnergy(
			&working.Lab, portal, &working.Planes[planeIndex], working.Observers, now, true, cfg,
		)
	case PortalActionSend:
		_, err = SendObserverWithLabEnergy(
			&working.Lab, portal, &working.Planes[planeIndex], working.Observers,
			now, true, availabilityRandom{}, cfg,
		)
	case PortalActionRecall:
		_, err = RecallObserverWithLabEnergy(
			&working.Lab, portal, &working.Planes[planeIndex], working.Observers,
			now, true, availabilityRandom{}, cfg,
		)
	default:
		return ActionAvailability{}, ErrSimulationInvariant
	}
	if err == nil {
		return ActionAvailability{Available: true}, nil
	}
	if isAvailabilityInvariant(err) || errors.Is(err, ErrConfirmationRequired) {
		return ActionAvailability{}, err
	}
	return ActionAvailability{Cause: err}, nil
}

func isAvailabilityInvariant(err error) bool {
	return errors.Is(err, ErrObserverInvariant) ||
		errors.Is(err, ErrLabEnergyInvariant) ||
		errors.Is(err, ErrExtractionInvariant) ||
		errors.Is(err, ErrSimulationInvariant) ||
		errors.Is(err, ErrEventInvariant)
}

type availabilityRandom struct{}

func (availabilityRandom) IntInclusive(min, _ int) int       { return min }
func (availabilityRandom) FloatRange(min, _ float64) float64 { return min }
