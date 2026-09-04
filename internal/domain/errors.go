package domain

import "errors"

// Stable domain errors. They carry no HTTP semantics; transport layers
// map them to status codes later (Final Spec §29, typical conflict 409).
var (
	ErrPortalNotOpen            = errors.New("portal is not open")
	ErrPortalAlreadyStable      = errors.New("portal is already stable")
	ErrPortalOverchargeRisk     = errors.New("stabilize rejected: portal energy above 85%")
	ErrConfirmationRequired     = errors.New("confirmation required")
	ErrNoFreePortalSlot         = errors.New("no free portal slot")
	ErrObserverNotAvailable     = errors.New("observer is not available")
	ErrObserverNotWaitingReturn = errors.New("observer is not waiting for return")
	ErrObserverLost             = errors.New("observer is lost")
	ErrObserverInvariant        = errors.New("observer lifecycle invariant violated")
	ErrPortalCriticalRisk       = errors.New("portal risk is critical")
	ErrPortalDirectionConflict  = errors.New("portal observer direction conflict")
	ErrPortalBusy               = errors.New("portal already has an observer in transit")
	ErrPortalCreaturesPresent   = errors.New("creatures are still inside portal")
	ErrNoAvailableObserver      = errors.New("no available observer")
	ErrNoWaitingObserver        = errors.New("no observer waiting in destination plane")
	ErrLabEnergyInvariant       = errors.New("laboratory energy invariant violated")
	ErrInsufficientLabEnergy    = errors.New("insufficient laboratory energy")
	ErrExtractionInvariant      = errors.New("extraction invariant violated")
	ErrExtractionSynchronizing  = errors.New("extraction portal is synchronizing")
)
