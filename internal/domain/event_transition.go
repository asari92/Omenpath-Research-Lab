package domain

import (
	"encoding/json"
	"sort"
	"time"

	"omenpath-lab/internal/config"
)

// EventsForStateTransition deterministically derives event drafts from one
// committed domain state transition. It performs no persistence and assigns
// no event IDs.
func EventsForStateTransition(
	before, after SimulationState,
	previousAt, now time.Time,
	cfg config.Config,
) ([]EventDraft, error) {
	if previousAt.IsZero() || now.IsZero() || previousAt.Location() != time.UTC ||
		now.Location() != time.UTC || now.Before(previousAt) || !validSimulationConfig(cfg) {
		return nil, ErrEventInvariant
	}

	events := make([]EventDraft, 0)
	beforePortals, err := portalsByID(before.Portals)
	if err != nil {
		return nil, err
	}
	afterPortals, err := portalsByID(after.Portals)
	if err != nil {
		return nil, err
	}

	collapses := make([]portalCollapse, 0)
	for id, portal := range afterPortals {
		prior, existed := beforePortals[id]
		if !existed {
			if portal.Status != PortalStatusOpen {
				continue
			}
			eventType := EventPortalOpened
			if portal.Kind == PortalKindExtraction {
				eventType = EventExtractionPortalOpened
			}
			events, err = appendEvent(events, eventType, &portal.ID, nil, &portal.DestinationPlaneID,
				"Portal opened", map[string]any{"kind": portal.Kind}, portal.OpenedAt)
			if err != nil {
				return nil, err
			}
			continue
		}

		if prior.Status == PortalStatusOpen && portal.Status != PortalStatusOpen {
			if portal.ClosedAt == nil {
				return nil, ErrEventInvariant
			}
			eventType := EventPortalClosed
			message := "Portal closed"
			if portal.Status == PortalStatusCollapsed {
				eventType = EventPortalCollapsed
				message = "Portal collapsed"
				collapses = append(collapses, portalCollapse{
					portalID: portal.ID,
					planeID:  portal.DestinationPlaneID,
					at:       *portal.ClosedAt,
				})
			}
			events, err = appendEvent(events, eventType, &portal.ID, nil, &portal.DestinationPlaneID,
				message, map[string]any{"reason": portal.TerminationReason}, *portal.ClosedAt)
			if err != nil {
				return nil, err
			}
		}

		if prior.Status == PortalStatusOpen && portal.Status == PortalStatusOpen &&
			prior.Stability == PortalUnstable && portal.Stability == PortalStable {
			events, err = appendEvent(events, EventPortalStabilized, &portal.ID, nil, &portal.DestinationPlaneID,
				"Portal stabilized", map[string]any{}, portal.UpdatedAt)
			if err != nil {
				return nil, err
			}
		}

		if prior.ExtractionSynchronizedAt == nil && portal.ExtractionSynchronizedAt != nil {
			events, err = appendEvent(events, EventExtractionSynchronized, &portal.ID, nil, &portal.DestinationPlaneID,
				"Extraction synchronized", map[string]any{}, *portal.ExtractionSynchronizedAt)
			if err != nil {
				return nil, err
			}
		}

		if prior.Status == PortalStatusOpen && portal.Status == PortalStatusOpen {
			previousLevel, previousOK := prior.RiskLevel(previousAt, cfg)
			currentLevel, currentOK := portal.RiskLevel(now, cfg)
			if previousOK && currentOK && previousLevel != currentLevel {
				events, err = appendEvent(events, EventRiskLevelChanged, &portal.ID, nil, &portal.DestinationPlaneID,
					"Portal risk level changed", map[string]any{"previous": previousLevel, "current": currentLevel}, now)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	overrideEvents, err := eventsForOverride(before.Lab, collapses, previousAt, now, cfg)
	if err != nil {
		return nil, err
	}
	events = append(events, overrideEvents...)

	observerEvents, err := eventsForObservers(before, after, beforePortals, afterPortals, now, cfg)
	if err != nil {
		return nil, err
	}
	events = append(events, observerEvents...)

	planeEvents, err := eventsForPlanes(before.Planes, after.Planes)
	if err != nil {
		return nil, err
	}
	events = append(events, planeEvents...)

	sort.SliceStable(events, func(i, j int) bool {
		return eventDraftLess(events[i], events[j])
	})
	return events, nil
}

func portalsByID(portals []Portal) (map[int64]Portal, error) {
	result := make(map[int64]Portal, len(portals))
	for _, portal := range portals {
		if portal.ID <= 0 {
			return nil, ErrEventInvariant
		}
		if _, duplicate := result[portal.ID]; duplicate {
			return nil, ErrEventInvariant
		}
		result[portal.ID] = portal
	}
	return result, nil
}

func eventsForOverride(
	before LabState,
	collapses []portalCollapse,
	previousAt, now time.Time,
	cfg config.Config,
) ([]EventDraft, error) {
	sort.Slice(collapses, func(i, j int) bool {
		if collapses[i].at.Equal(collapses[j].at) {
			return collapses[i].portalID < collapses[j].portalID
		}
		return collapses[i].at.Before(collapses[j].at)
	})
	events := make([]EventDraft, 0, len(collapses)+1)
	deadline := cloneTime(before.LeylineOverrideUntil)
	windowStart := previousAt
	for _, collapse := range collapses {
		collapseAt := collapse.at
		if deadline != nil && windowStart.Before(*deadline) && !collapseAt.Before(*deadline) {
			var err error
			events, err = appendEvent(events, EventLeylineOverrideEnded, nil, nil, nil,
				"Leyline Override ended", map[string]any{}, *deadline)
			if err != nil {
				return nil, err
			}
		}
		newDeadline := collapseAt.Add(cfg.EmergencyDuration)
		var err error
		events, err = appendEvent(events, EventLeylineOverrideStarted, &collapse.portalID, nil, &collapse.planeID,
			"Leyline Override started", map[string]any{"until": newDeadline}, collapseAt)
		if err != nil {
			return nil, err
		}
		deadline = &newDeadline
		windowStart = collapseAt
	}
	if deadline != nil && windowStart.Before(*deadline) && !now.Before(*deadline) {
		var err error
		events, err = appendEvent(events, EventLeylineOverrideEnded, nil, nil, nil,
			"Leyline Override ended", map[string]any{}, *deadline)
		if err != nil {
			return nil, err
		}
	}
	return events, nil
}

type portalCollapse struct {
	portalID int64
	planeID  int64
	at       time.Time
}

func eventsForObservers(
	before, after SimulationState,
	beforePortals, afterPortals map[int64]Portal,
	now time.Time,
	cfg config.Config,
) ([]EventDraft, error) {
	beforeObservers, err := observersByID(before.Observers)
	if err != nil {
		return nil, err
	}
	afterObservers, err := observersByID(after.Observers)
	if err != nil {
		return nil, err
	}
	events := make([]EventDraft, 0)
	for id, observer := range afterObservers {
		prior, existed := beforeObservers[id]
		if !existed || prior.Status == observer.Status && sameObserverPhase(prior, observer) {
			continue
		}

		if prior.Status == ObserverAvailable && observer.Status == ObserverOutbound {
			planeID := portalPlaneID(afterPortals, observer.ActivePortalID)
			events, err = appendEvent(events, EventObserverDispatched, observer.ActivePortalID, &observer.ID, planeID,
				"Observer dispatched", map[string]any{}, requiredTime(observer.PhaseStartedAt))
			if err != nil {
				return nil, err
			}
		}

		if prior.Status == ObserverOutbound && observer.Status != ObserverOutbound && observer.Status != ObserverLost {
			arrivalAt := requiredTime(prior.PhaseEndsAt)
			planeID := observer.CurrentPlaneID
			if planeID == nil {
				planeID = portalPlaneID(beforePortals, prior.ActivePortalID)
			}
			events, err = appendEvent(events, EventObserverArrived, prior.ActivePortalID, &observer.ID, planeID,
				"Observer arrived", map[string]any{}, arrivalAt)
			if err != nil {
				return nil, err
			}
			events, err = appendEvent(events, EventResearchStarted, prior.ActivePortalID, &observer.ID, planeID,
				"Research started", map[string]any{}, arrivalAt)
			if err != nil {
				return nil, err
			}
			if observer.Status != ObserverExploring {
				completedAt := arrivalAt.Add(cfg.ResearchDuration)
				events, err = appendEvent(events, EventResearchCompleted, nil, &observer.ID, planeID,
					"Research completed", map[string]any{}, completedAt)
				if err != nil {
					return nil, err
				}
			}
		} else if prior.Status == ObserverExploring && observer.Status != ObserverExploring && observer.Status != ObserverLost {
			events, err = appendEvent(events, EventResearchCompleted, nil, &observer.ID, prior.CurrentPlaneID,
				"Research completed", map[string]any{}, requiredTime(prior.PhaseEndsAt))
			if err != nil {
				return nil, err
			}
		}

		if (prior.Status == ObserverWaitingReturn || prior.Status == ObserverExploring || prior.Status == ObserverOutbound) &&
			observer.Status == ObserverReturning {
			events, err = appendEvent(events, EventObserverReturnStarted, observer.ActivePortalID, &observer.ID, observer.CurrentPlaneID,
				"Observer return started", map[string]any{}, requiredTime(observer.PhaseStartedAt))
			if err != nil {
				return nil, err
			}
		}

		if prior.Status == ObserverReturning && observer.Status == ObserverAvailable {
			events, err = appendEvent(events, EventObserverReturned, prior.ActivePortalID, &observer.ID, prior.CurrentPlaneID,
				"Observer returned", map[string]any{}, requiredTime(prior.PhaseEndsAt))
			if err != nil {
				return nil, err
			}
		}

		if observer.Status == ObserverLost && prior.Status != ObserverLost {
			lostAt := observer.UpdatedAt
			if lostAt.IsZero() || lostAt.After(now) {
				return nil, ErrEventInvariant
			}
			planeID := prior.CurrentPlaneID
			if planeID == nil {
				planeID = portalPlaneID(beforePortals, prior.ActivePortalID)
			}
			events, err = appendEvent(events, EventObserverLost, prior.ActivePortalID, &observer.ID, planeID,
				"Observer lost", map[string]any{}, lostAt)
			if err != nil {
				return nil, err
			}
		}
	}
	return events, nil
}

func observersByID(observers []Observer) (map[int64]Observer, error) {
	result := make(map[int64]Observer, len(observers))
	for _, observer := range observers {
		if observer.ID <= 0 {
			return nil, ErrEventInvariant
		}
		if _, duplicate := result[observer.ID]; duplicate {
			return nil, ErrEventInvariant
		}
		result[observer.ID] = observer
	}
	return result, nil
}

func sameObserverPhase(a, b Observer) bool {
	return equalInt64Pointer(a.CurrentPlaneID, b.CurrentPlaneID) &&
		equalInt64Pointer(a.ActivePortalID, b.ActivePortalID) &&
		equalTimePointer(a.PhaseStartedAt, b.PhaseStartedAt) &&
		equalTimePointer(a.PhaseEndsAt, b.PhaseEndsAt)
}

func portalPlaneID(portals map[int64]Portal, portalID *int64) *int64 {
	if portalID == nil {
		return nil
	}
	portal, ok := portals[*portalID]
	if !ok {
		return nil
	}
	return &portal.DestinationPlaneID
}

func eventsForPlanes(before, after []Plane) ([]EventDraft, error) {
	beforePlanes, err := planesByID(before)
	if err != nil {
		return nil, err
	}
	if _, err := planesByID(after); err != nil {
		return nil, err
	}
	events := make([]EventDraft, 0)
	for _, plane := range after {
		prior, existed := beforePlanes[plane.ID]
		if !existed || prior.Explored || !plane.Explored || plane.ExploredAt == nil {
			continue
		}
		var err error
		events, err = appendEvent(events, EventPlaneExplored, nil, nil, &plane.ID,
			"Plane explored", map[string]any{}, *plane.ExploredAt)
		if err != nil {
			return nil, err
		}
	}
	return events, nil
}

func planesByID(planes []Plane) (map[int64]Plane, error) {
	result := make(map[int64]Plane, len(planes))
	for _, plane := range planes {
		if plane.ID <= 0 {
			return nil, ErrEventInvariant
		}
		if _, duplicate := result[plane.ID]; duplicate {
			return nil, ErrEventInvariant
		}
		result[plane.ID] = plane
	}
	return result, nil
}

func appendEvent(
	events []EventDraft,
	eventType EventType,
	portalID, observerID, planeID *int64,
	message string,
	payload map[string]any,
	createdAt time.Time,
) ([]EventDraft, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrEventInvariant
	}
	draft := EventDraft{
		EventType:   eventType,
		PortalID:    cloneInt64(portalID),
		ObserverID:  cloneInt64(observerID),
		PlaneID:     cloneInt64(planeID),
		Message:     message,
		PayloadJSON: string(encoded),
		CreatedAt:   createdAt,
	}
	if err := draft.Validate(); err != nil {
		return nil, err
	}
	return append(events, draft), nil
}

func eventDraftLess(a, b EventDraft) bool {
	if !a.CreatedAt.Equal(b.CreatedAt) {
		return a.CreatedAt.Before(b.CreatedAt)
	}
	if eventTieRank(a.EventType) != eventTieRank(b.EventType) {
		return eventTieRank(a.EventType) < eventTieRank(b.EventType)
	}
	if pointerValue(a.PortalID) != pointerValue(b.PortalID) {
		return pointerValue(a.PortalID) < pointerValue(b.PortalID)
	}
	if pointerValue(a.ObserverID) != pointerValue(b.ObserverID) {
		return pointerValue(a.ObserverID) < pointerValue(b.ObserverID)
	}
	if pointerValue(a.PlaneID) != pointerValue(b.PlaneID) {
		return pointerValue(a.PlaneID) < pointerValue(b.PlaneID)
	}
	return a.EventType < b.EventType
}

func eventTieRank(eventType EventType) int {
	switch eventType {
	case EventPortalClosed, EventPortalCollapsed:
		return 10
	case EventLeylineOverrideEnded:
		return 20
	case EventLeylineOverrideStarted:
		return 21
	case EventPortalStabilized, EventObserverDispatched, EventObserverArrived:
		return 30
	case EventResearchStarted:
		return 31
	case EventResearchCompleted:
		return 40
	case EventExtractionSynchronized:
		return 50
	case EventObserverReturnStarted:
		return 51
	case EventObserverReturned, EventObserverLost:
		return 60
	case EventPlaneExplored:
		return 61
	case EventRiskLevelChanged:
		return 70
	case EventPortalOpened, EventExtractionPortalOpened:
		return 80
	default:
		return 90
	}
}

func pointerValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func requiredTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func equalInt64Pointer(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func equalTimePointer(a, b *time.Time) bool {
	return a == nil && b == nil || a != nil && b != nil && a.Equal(*b)
}
