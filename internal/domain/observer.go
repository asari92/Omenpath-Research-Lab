package domain

import "time"

// ObserverStatus (Final Spec §15). LOST is terminal.
type ObserverStatus string

const (
	ObserverAvailable     ObserverStatus = "AVAILABLE"
	ObserverOutbound      ObserverStatus = "OUTBOUND"
	ObserverExploring     ObserverStatus = "EXPLORING"
	ObserverWaitingReturn ObserverStatus = "WAITING_RETURN"
	ObserverReturning     ObserverStatus = "RETURNING"
	ObserverLost          ObserverStatus = "LOST"
)

// Observer is a permanent entity with current-state fields only
// (Final Spec §15). Past travel history lives in the Event Log.
type Observer struct {
	ID             int64
	Status         ObserverStatus
	CurrentPlaneID *int64 // nil = Laboratory (§15)
	ActivePortalID *int64 // set only during transit
	PhaseStartedAt *time.Time
	PhaseEndsAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewObserver creates an Observer in the canonical AVAILABLE state in the
// Laboratory (Final Spec §15).
func NewObserver(id int64, now time.Time) Observer {
	return Observer{
		ID:        id,
		Status:    ObserverAvailable,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// NewObserverRoster creates count permanent Observer records with stable,
// one-based IDs. The configured production count is supplied by the caller.
func NewObserverRoster(count int, now time.Time) []Observer {
	roster := make([]Observer, count)
	for i := range roster {
		roster[i] = NewObserver(int64(i+1), now)
	}
	return roster
}

// IsTerminal reports whether the Observer is permanently LOST.
func (o Observer) IsTerminal() bool {
	return o.Status == ObserverLost
}
