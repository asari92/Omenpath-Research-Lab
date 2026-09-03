package domain

import "time"

// EventType values (Final Spec §26).
type EventType string

const (
	EventPortalOpened           EventType = "PORTAL_OPENED"
	EventPortalStabilized       EventType = "PORTAL_STABILIZED"
	EventPortalClosed           EventType = "PORTAL_CLOSED"
	EventPortalCollapsed        EventType = "PORTAL_COLLAPSED"
	EventRiskLevelChanged       EventType = "RISK_LEVEL_CHANGED"
	EventObserverDispatched     EventType = "OBSERVER_DISPATCHED"
	EventObserverArrived        EventType = "OBSERVER_ARRIVED"
	EventResearchStarted        EventType = "RESEARCH_STARTED"
	EventResearchCompleted      EventType = "RESEARCH_COMPLETED"
	EventObserverReturnStarted  EventType = "OBSERVER_RETURN_STARTED"
	EventObserverReturned       EventType = "OBSERVER_RETURNED"
	EventObserverLost           EventType = "OBSERVER_LOST"
	EventPlaneExplored          EventType = "PLANE_EXPLORED"
	EventExtractionPortalOpened EventType = "EXTRACTION_PORTAL_OPENED"
	EventExtractionSynchronized EventType = "EXTRACTION_SYNCHRONIZED"
	EventLeylineOverrideStarted EventType = "LEYLINE_OVERRIDE_STARTED"
	EventLeylineOverrideEnded   EventType = "LEYLINE_OVERRIDE_ENDED"
	EventActionRejected         EventType = "ACTION_REJECTED"
)

// Event is the shared source for Portal History and the global Event Log
// (Final Spec §25, §26).
type Event struct {
	ID          int64
	EventType   EventType
	PortalID    *int64
	ObserverID  *int64
	PlaneID     *int64
	Message     string
	PayloadJSON string
	CreatedAt   time.Time
}
