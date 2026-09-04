package domain

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
)

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

// EventDraft is a validated Event before persistence assigns its positive ID.
type EventDraft struct {
	EventType   EventType
	PortalID    *int64
	ObserverID  *int64
	PlaneID     *int64
	Message     string
	PayloadJSON string
	CreatedAt   time.Time
}

// IsEventType reports whether value belongs to the closed event enum.
func IsEventType(value EventType) bool {
	switch value {
	case EventPortalOpened,
		EventPortalStabilized,
		EventPortalClosed,
		EventPortalCollapsed,
		EventRiskLevelChanged,
		EventObserverDispatched,
		EventObserverArrived,
		EventResearchStarted,
		EventResearchCompleted,
		EventObserverReturnStarted,
		EventObserverReturned,
		EventObserverLost,
		EventPlaneExplored,
		EventExtractionPortalOpened,
		EventExtractionSynchronized,
		EventLeylineOverrideStarted,
		EventLeylineOverrideEnded,
		EventActionRejected:
		return true
	default:
		return false
	}
}

// Validate enforces the storage-independent event invariants.
func (d EventDraft) Validate() error {
	if !IsEventType(d.EventType) || strings.TrimSpace(d.Message) == "" ||
		d.CreatedAt.IsZero() || d.CreatedAt.Location() != time.UTC ||
		!validEventEntityID(d.PortalID) ||
		!validEventEntityID(d.ObserverID) ||
		!validEventEntityID(d.PlaneID) ||
		!isJSONObject(d.PayloadJSON) {
		return ErrEventInvariant
	}
	return nil
}

func validEventEntityID(id *int64) bool {
	return id == nil || *id > 0
}

func isJSONObject(payload string) bool {
	var object map[string]json.RawMessage
	return json.Unmarshal([]byte(payload), &object) == nil && object != nil
}

// ValidatePersisted requires both valid event content and a persistence ID.
func (e Event) ValidatePersisted() error {
	if e.ID <= 0 {
		return ErrEventInvariant
	}
	return (EventDraft{
		EventType:   e.EventType,
		PortalID:    e.PortalID,
		ObserverID:  e.ObserverID,
		PlaneID:     e.PlaneID,
		Message:     e.Message,
		PayloadJSON: e.PayloadJSON,
		CreatedAt:   e.CreatedAt,
	}).Validate()
}

// PortalHistory returns the shared event source filtered by exact portal ID,
// ordered chronologically. Pointer fields are deep-copied for caller safety.
func PortalHistory(events []Event, portalID int64) []Event {
	history := make([]Event, 0)
	for _, event := range events {
		if event.PortalID == nil || *event.PortalID != portalID {
			continue
		}
		history = append(history, cloneEvent(event))
	}
	sort.SliceStable(history, func(i, j int) bool {
		if history[i].CreatedAt.Equal(history[j].CreatedAt) {
			return history[i].ID < history[j].ID
		}
		return history[i].CreatedAt.Before(history[j].CreatedAt)
	})
	return history
}

func cloneEvent(event Event) Event {
	event.PortalID = cloneInt64(event.PortalID)
	event.ObserverID = cloneInt64(event.ObserverID)
	event.PlaneID = cloneInt64(event.PlaneID)
	return event
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

// NewActionRejectedEvent describes a domain-level rejected action without
// coupling the domain event to transport error semantics.
func NewActionRejectedEvent(
	now time.Time,
	action string,
	portalID, observerID, planeID *int64,
	cause error,
) (EventDraft, error) {
	if strings.TrimSpace(action) == "" || cause == nil {
		return EventDraft{}, ErrEventInvariant
	}
	causeText := cause.Error()
	payload, err := json.Marshal(map[string]string{
		"action": action,
		"cause":  causeText,
	})
	if err != nil {
		return EventDraft{}, ErrEventInvariant
	}
	draft := EventDraft{
		EventType:   EventActionRejected,
		PortalID:    cloneInt64(portalID),
		ObserverID:  cloneInt64(observerID),
		PlaneID:     cloneInt64(planeID),
		Message:     "Action rejected: " + causeText,
		PayloadJSON: string(payload),
		CreatedAt:   now,
	}
	if err := draft.Validate(); err != nil {
		return EventDraft{}, err
	}
	return draft, nil
}
