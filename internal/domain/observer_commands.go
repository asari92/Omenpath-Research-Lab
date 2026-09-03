package domain

import (
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// AvailableObserverIndex returns the slice index of the AVAILABLE Observer
// with the lowest ID. The stable choice keeps command results deterministic.
func AvailableObserverIndex(observers []Observer) (index int, ok bool) {
	selected := -1
	for i := range observers {
		if observers[i].Status != ObserverAvailable {
			continue
		}
		if selected == -1 || observers[i].ID < observers[selected].ID {
			selected = i
		}
	}
	return selected, selected != -1
}

// LongestWaitingObserverIndex returns the longest-waiting Observer in planeID.
// Exact waiting-time ties are resolved by the lowest Observer ID.
func LongestWaitingObserverIndex(observers []Observer, planeID int64) (index int, ok bool, err error) {
	selected := -1
	for i := range observers {
		observer := &observers[i]
		if observer.Status != ObserverWaitingReturn {
			continue
		}
		if observer.CurrentPlaneID == nil || observer.PhaseStartedAt == nil {
			return -1, false, ErrObserverInvariant
		}
		if *observer.CurrentPlaneID != planeID {
			continue
		}
		if selected == -1 || observer.PhaseStartedAt.Before(*observers[selected].PhaseStartedAt) ||
			(observer.PhaseStartedAt.Equal(*observers[selected].PhaseStartedAt) && observer.ID < observers[selected].ID) {
			selected = i
		}
	}
	return selected, selected != -1, nil
}

// ActiveTransitObserverIndex returns the single Observer actively traversing
// portalID. A stale or duplicate transit is an aggregate invariant violation.
func ActiveTransitObserverIndex(observers []Observer, portalID int64, now time.Time) (index int, ok bool, err error) {
	selected := -1
	for i := range observers {
		observer := &observers[i]
		if observer.Status != ObserverOutbound && observer.Status != ObserverReturning {
			continue
		}
		if observer.ActivePortalID == nil || observer.PhaseStartedAt == nil || observer.PhaseEndsAt == nil {
			return -1, false, ErrObserverInvariant
		}
		if observer.Status == ObserverOutbound && observer.CurrentPlaneID != nil {
			return -1, false, ErrObserverInvariant
		}
		if observer.Status == ObserverReturning && observer.CurrentPlaneID == nil {
			return -1, false, ErrObserverInvariant
		}
		if *observer.ActivePortalID != portalID {
			continue
		}
		if !observer.PhaseStartedAt.Before(*observer.PhaseEndsAt) || !now.Before(*observer.PhaseEndsAt) {
			return -1, false, ErrObserverInvariant
		}
		if selected != -1 {
			return -1, false, ErrObserverInvariant
		}
		selected = i
	}
	return selected, selected != -1, nil
}

// SendObserver starts an admitted Lab-to-Plane transit and permanently fixes
// an unused Portal to the OUTBOUND direction.
func SendObserver(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	_ = plane

	if portal.Status != PortalStatusOpen {
		return 0, ErrPortalNotOpen
	}
	risk, _ := portal.RiskLevel(now, cfg)
	if risk == RiskCritical {
		return 0, ErrPortalCriticalRisk
	}
	if portal.ObserverFlow == PortalFlowInbound {
		return 0, ErrPortalDirectionConflict
	}
	if portal.CreaturesInside(now, cfg) > 0 {
		return 0, ErrPortalCreaturesPresent
	}
	_, busy, err := ActiveTransitObserverIndex(observers, portal.ID, now)
	if err != nil {
		return 0, err
	}
	if busy {
		return 0, ErrPortalBusy
	}

	index, ok := AvailableObserverIndex(observers)
	if !ok {
		return 0, ErrNoAvailableObserver
	}
	if portal.Stability == PortalUnstable && !confirmUnstable {
		return 0, ErrConfirmationRequired
	}
	if err = observers[index].StartOutbound(now, portal.ID, rnd, cfg); err != nil {
		return 0, err
	}
	if portal.ObserverFlow == PortalFlowNone {
		portal.ObserverFlow = PortalFlowOutbound
		portal.UpdatedAt = now
	}
	return observers[index].ID, nil
}

// RecallObserver starts an admitted Plane-to-Lab transit and permanently
// fixes an unused Portal to the INBOUND direction.
func RecallObserver(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	_ = plane

	if portal.Status != PortalStatusOpen {
		return 0, ErrPortalNotOpen
	}
	risk, _ := portal.RiskLevel(now, cfg)
	if risk == RiskCritical {
		return 0, ErrPortalCriticalRisk
	}
	if portal.ObserverFlow == PortalFlowOutbound {
		return 0, ErrPortalDirectionConflict
	}
	if portal.CreaturesInside(now, cfg) > 0 {
		return 0, ErrPortalCreaturesPresent
	}
	_, busy, err := ActiveTransitObserverIndex(observers, portal.ID, now)
	if err != nil {
		return 0, err
	}
	if busy {
		return 0, ErrPortalBusy
	}

	index, ok, err := LongestWaitingObserverIndex(observers, portal.DestinationPlaneID)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, ErrNoWaitingObserver
	}
	if portal.Stability == PortalUnstable && !confirmUnstable {
		return 0, ErrConfirmationRequired
	}
	if err = observers[index].StartReturning(now, portal.ID, rnd, cfg); err != nil {
		return 0, err
	}
	if portal.ObserverFlow == PortalFlowNone {
		portal.ObserverFlow = PortalFlowInbound
		portal.UpdatedAt = now
	}
	return observers[index].ID, nil
}
