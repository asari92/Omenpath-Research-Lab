package transport

import (
	"errors"

	"omenpath-lab/internal/domain"
)

type ErrorDescriptor struct {
	Code        string
	Message     string
	Confirmable bool
}

func DescribeDomainError(err error) (ErrorDescriptor, bool) {
	known := []struct {
		err         error
		code        string
		confirmable bool
	}{
		{domain.ErrPortalNotOpen, "PORTAL_NOT_OPEN", false},
		{domain.ErrPortalAlreadyStable, "PORTAL_ALREADY_STABLE", false},
		{domain.ErrPortalOverchargeRisk, "PORTAL_OVERCHARGE_RISK", false},
		{domain.ErrConfirmationRequired, "CONFIRMATION_REQUIRED", true},
		{domain.ErrNoFreePortalSlot, "NO_FREE_PORTAL_SLOT", false},
		{domain.ErrObserverNotAvailable, "OBSERVER_NOT_AVAILABLE", false},
		{domain.ErrObserverNotWaitingReturn, "OBSERVER_NOT_WAITING_RETURN", false},
		{domain.ErrObserverLost, "OBSERVER_LOST", false},
		{domain.ErrPortalCriticalRisk, "PORTAL_CRITICAL_RISK", false},
		{domain.ErrPortalDirectionConflict, "PORTAL_DIRECTION_CONFLICT", false},
		{domain.ErrPortalBusy, "PORTAL_BUSY", false},
		{domain.ErrPortalCreaturesPresent, "PORTAL_CREATURES_PRESENT", false},
		{domain.ErrNoAvailableObserver, "NO_AVAILABLE_OBSERVER", false},
		{domain.ErrNoWaitingObserver, "NO_WAITING_OBSERVER", false},
		{domain.ErrInsufficientLabEnergy, "INSUFFICIENT_LAB_ENERGY", false},
		{domain.ErrExtractionSynchronizing, "EXTRACTION_SYNCHRONIZING", false},
	}
	for _, candidate := range known {
		if errors.Is(err, candidate.err) {
			return ErrorDescriptor{
				Code: candidate.code, Message: candidate.err.Error(), Confirmable: candidate.confirmable,
			}, true
		}
	}
	return ErrorDescriptor{}, false
}
