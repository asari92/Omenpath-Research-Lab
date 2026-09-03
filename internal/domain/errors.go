package domain

import "errors"

// Stable domain errors. They carry no HTTP semantics; transport layers
// map them to status codes later (Final Spec §29, typical conflict 409).
var (
	ErrPortalNotOpen        = errors.New("portal is not open")
	ErrPortalAlreadyStable  = errors.New("portal is already stable")
	ErrPortalOverchargeRisk = errors.New("stabilize rejected: portal energy above 85%")
	ErrConfirmationRequired = errors.New("confirmation required")
	ErrNoFreePortalSlot     = errors.New("no free portal slot")
	ErrObserverNotAvailable = errors.New("observer is not available")
	ErrObserverLost         = errors.New("observer is lost")
	ErrObserverInvariant    = errors.New("observer lifecycle invariant violated")
)
