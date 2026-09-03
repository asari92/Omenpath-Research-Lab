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
