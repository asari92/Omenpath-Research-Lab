package domain

import "time"

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
