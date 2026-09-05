// Package transport defines the sole public JSON view models shared by REST
// and realtime delivery. Domain-only and hidden scheduling fields never cross
// this boundary.
package transport

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
)

type AppDTO struct {
	Mode               domain.AppMode                 `json:"mode"`
	TutorialStep       int                            `json:"tutorial_step"`
	TutorialPhase      domain.TutorialPhase           `json:"tutorial_phase"`
	TutorialPortalID   *int64                         `json:"tutorial_portal_id"`
	TutorialPlaneID    *int64                         `json:"tutorial_plane_id"`
	TutorialObserverID *int64                         `json:"tutorial_observer_id"`
	ExpectedAction     *domain.TutorialExpectedAction `json:"expected_action"`
}

type LabDTO struct {
	CurrentEnergy         int        `json:"current_energy"`
	MaximumEnergy         int        `json:"maximum_energy"`
	LeylineOverrideActive bool       `json:"leyline_override_active"`
	LeylineOverrideUntil  *time.Time `json:"leyline_override_until"`
}

type ExplorationDTO struct {
	Explored int `json:"explored"`
	Total    int `json:"total"`
}

type ObserverCountsDTO struct {
	Available     int `json:"available"`
	Outbound      int `json:"outbound"`
	Exploring     int `json:"exploring"`
	WaitingReturn int `json:"waiting_return"`
	Returning     int `json:"returning"`
	Lost          int `json:"lost"`
	InLab         int `json:"in_lab"`
	InWorlds      int `json:"in_worlds"`
	InTransit     int `json:"in_transit"`
}

type ObserverTransitDTO struct {
	ObserverID       int64                 `json:"observer_id"`
	PortalID         int64                 `json:"portal_id"`
	Direction        domain.ObserverStatus `json:"direction"`
	StartedAt        time.Time             `json:"started_at"`
	CompletesAt      time.Time             `json:"completes_at"`
	RemainingSeconds int64                 `json:"remaining_seconds"`
}

type PortalCountsDTO struct {
	Active    int `json:"active"`
	Maximum   int `json:"maximum"`
	Critical  int `json:"critical"`
	Closed    int `json:"closed"`
	Collapsed int `json:"collapsed"`
}

type PlaneDTO struct {
	ID                     int64      `json:"id"`
	Name                   string     `json:"name"`
	Aliases                []string   `json:"aliases"`
	CatalogTier            string     `json:"catalog_tier"`
	Explored               bool       `json:"explored"`
	ExploredAt             *time.Time `json:"explored_at"`
	ObserversInPlane       int        `json:"observers_in_plane"`
	ObserversWaitingReturn int        `json:"observers_waiting_return"`
}

type QuickActionsDTO struct {
	CanStabilize               bool    `json:"can_stabilize"`
	StabilizeUnavailableReason *string `json:"stabilize_unavailable_reason"`
	CanClose                   bool    `json:"can_close"`
	CloseUnavailableReason     *string `json:"close_unavailable_reason"`
	CanSend                    bool    `json:"can_send_observer"`
	SendUnavailableReason      *string `json:"send_observer_unavailable_reason"`
	CanRecall                  bool    `json:"can_recall_observer"`
	RecallUnavailableReason    *string `json:"recall_observer_unavailable_reason"`
}

type SlotPortalDTO struct {
	ID                   int64                  `json:"id"`
	Name                 string                 `json:"name"`
	DestinationPlaneID   int64                  `json:"destination_plane_id"`
	DestinationPlaneName string                 `json:"destination_plane_name"`
	DestinationExplored  bool                   `json:"destination_explored"`
	Energy               float64                `json:"energy"`
	Stability            domain.PortalStability `json:"stability"`
	TimeRemainingSeconds int64                  `json:"time_remaining_seconds"`
	CreaturesInside      int                    `json:"creatures_inside"`
	Status               domain.PortalStatus    `json:"status"`
	QuickActions         QuickActionsDTO        `json:"quick_actions"`
}

type SlotDTO struct {
	SlotIndex int            `json:"slot_index"`
	Portal    *SlotPortalDTO `json:"portal"`
}

type StateSnapshot struct {
	GeneratedAt            time.Time            `json:"generated_at"`
	App                    AppDTO               `json:"app"`
	Lab                    LabDTO               `json:"lab"`
	Exploration            ExplorationDTO       `json:"exploration"`
	Observers              ObserverCountsDTO    `json:"observers"`
	ObserverTransits       []ObserverTransitDTO `json:"observer_transits"`
	Portals                PortalCountsDTO      `json:"portals"`
	NeedsAttentionPortalID *int64               `json:"needs_attention_portal_id"`
	Slots                  []SlotDTO            `json:"slots"`
	Planes                 []PlaneDTO           `json:"planes"`
}

type PortalViewDTO struct {
	ID                   int64                    `json:"id"`
	Name                 string                   `json:"name"`
	SlotIndex            int                      `json:"slot_index"`
	Kind                 domain.PortalKind        `json:"kind"`
	Status               domain.PortalStatus      `json:"status"`
	TerminationReason    domain.TerminationReason `json:"termination_reason"`
	Energy               float64                  `json:"energy"`
	Stability            domain.PortalStability   `json:"stability"`
	TimeRemainingSeconds int64                    `json:"time_remaining_seconds"`
	CreaturesInside      int                      `json:"creatures_inside"`
	ObserverFlow         domain.PortalFlow        `json:"observer_flow"`
	OpenedAt             time.Time                `json:"opened_at"`
	ClosedAt             *time.Time               `json:"closed_at"`
	QuickActions         QuickActionsDTO          `json:"quick_actions"`
}

type DestinationDTO struct {
	PlaneID                 int64    `json:"plane_id"`
	Name                    string   `json:"name"`
	Aliases                 []string `json:"aliases"`
	CatalogTier             string   `json:"catalog_tier"`
	Explored                bool     `json:"explored"`
	ObserversExploring      int      `json:"observers_exploring"`
	ObserversWaitingReturn  int      `json:"observers_waiting_return"`
	PreviousConnectionCount int      `json:"previous_connection_count"`
}

type EventDTO struct {
	ID          int64            `json:"id"`
	EventType   domain.EventType `json:"event_type"`
	PortalID    *int64           `json:"portal_id"`
	ObserverID  *int64           `json:"observer_id"`
	PlaneID     *int64           `json:"plane_id"`
	Message     string           `json:"message"`
	PayloadJSON json.RawMessage  `json:"payload_json"`
	CreatedAt   time.Time        `json:"created_at"`
}

type PortalDetails struct {
	ObserverTransit *ObserverTransitDTO    `json:"observer_transit"`
	GeneratedAt     time.Time              `json:"generated_at"`
	Portal          PortalViewDTO          `json:"portal"`
	Destination     DestinationDTO         `json:"destination"`
	RiskLevel       *domain.RiskLevel      `json:"risk_level"`
	Recommendation  *domain.Recommendation `json:"recommendation"`
	History         []EventDTO             `json:"history"`
}

func BuildStateSnapshot(snapshot persistence.Snapshot, now time.Time, cfg config.Config) (StateSnapshot, error) {
	now = now.UTC()
	transits, err := BuildObserverTransits(snapshot.Simulation, now)
	if err != nil {
		return StateSnapshot{}, err
	}
	planes := make(map[int64]domain.Plane, len(snapshot.Simulation.Planes))
	planeIndexes := make(map[int64]int, len(snapshot.Simulation.Planes))
	result := StateSnapshot{
		GeneratedAt:      now,
		ObserverTransits: transits,
		App: AppDTO{
			Mode: snapshot.App.Mode, TutorialStep: snapshot.App.TutorialStep,
			TutorialPhase: snapshot.App.TutorialPhase, TutorialPortalID: cloneInt64(snapshot.App.TutorialPortalID),
			TutorialPlaneID: cloneInt64(snapshot.App.TutorialPlaneID), TutorialObserverID: cloneInt64(snapshot.App.TutorialObserverID),
		},
		Lab: LabDTO{
			CurrentEnergy: snapshot.Simulation.Lab.CurrentEnergy(now, cfg), MaximumEnergy: cfg.LabEnergyMax,
			LeylineOverrideActive: snapshot.Simulation.Lab.LeylineOverrideActive(now, cfg),
			LeylineOverrideUntil:  cloneTime(snapshot.Simulation.Lab.LeylineOverrideUntil),
		},
		Exploration: ExplorationDTO{Total: len(snapshot.Simulation.Planes)},
		Portals:     PortalCountsDTO{Maximum: cfg.MaxActivePortals},
		Slots:       make([]SlotDTO, cfg.MaxActivePortals),
		Planes:      make([]PlaneDTO, 0, len(snapshot.Simulation.Planes)),
	}
	if snapshot.App.Mode == domain.ModeTutorial {
		expected, err := domain.ExpectedTutorialAction(snapshot.App)
		if err != nil {
			return StateSnapshot{}, err
		}
		result.App.ExpectedAction = &expected
	}
	for i, plane := range snapshot.Simulation.Planes {
		if plane.ID <= 0 {
			return StateSnapshot{}, domain.ErrSimulationInvariant
		}
		planes[plane.ID] = plane
		planeIndexes[plane.ID] = len(result.Planes)
		if plane.Explored {
			result.Exploration.Explored++
		}
		result.Planes = append(result.Planes, buildPlane(plane))
		_ = i
	}
	for _, observer := range snapshot.Simulation.Observers {
		countObserver(&result.Observers, observer.Status)
		if observer.CurrentPlaneID == nil {
			continue
		}
		index, ok := planeIndexes[*observer.CurrentPlaneID]
		if !ok {
			return StateSnapshot{}, domain.ErrSimulationInvariant
		}
		switch observer.Status {
		case domain.ObserverExploring, domain.ObserverWaitingReturn, domain.ObserverReturning:
			result.Planes[index].ObserversInPlane++
			if observer.Status == domain.ObserverWaitingReturn {
				result.Planes[index].ObserversWaitingReturn++
			}
		}
	}
	for i := range result.Slots {
		result.Slots[i].SlotIndex = i + 1
	}
	for _, portal := range snapshot.Simulation.Portals {
		switch portal.Status {
		case domain.PortalStatusOpen:
			result.Portals.Active++
			risk, _ := portal.RiskLevel(now, cfg)
			if risk == domain.RiskCritical {
				result.Portals.Critical++
			}
			if portal.SlotIndex < 1 || portal.SlotIndex > len(result.Slots) || result.Slots[portal.SlotIndex-1].Portal != nil {
				return StateSnapshot{}, domain.ErrSimulationInvariant
			}
			plane, ok := planes[portal.DestinationPlaneID]
			if !ok {
				return StateSnapshot{}, domain.ErrSimulationInvariant
			}
			view, err := buildSlotPortal(snapshot.Simulation, portal, plane, now, cfg)
			if err != nil {
				return StateSnapshot{}, err
			}
			result.Slots[portal.SlotIndex-1].Portal = &view
		case domain.PortalStatusClosed:
			result.Portals.Closed++
		case domain.PortalStatusCollapsed:
			result.Portals.Collapsed++
		default:
			return StateSnapshot{}, domain.ErrSimulationInvariant
		}
	}
	if index, ok, err := domain.NeedsAttentionPortalIndex(snapshot.Simulation.Portals, now, cfg); err != nil {
		return StateSnapshot{}, err
	} else if ok {
		id := snapshot.Simulation.Portals[index].ID
		result.NeedsAttentionPortalID = &id
	}
	return result, nil
}

func BuildPortalDetails(snapshot persistence.Snapshot, portalID int64, history []domain.Event, now time.Time, cfg config.Config) (PortalDetails, error) {
	now = now.UTC()
	transits, err := BuildObserverTransits(snapshot.Simulation, now)
	if err != nil {
		return PortalDetails{}, err
	}
	var portal *domain.Portal
	for i := range snapshot.Simulation.Portals {
		if snapshot.Simulation.Portals[i].ID == portalID {
			candidate := snapshot.Simulation.Portals[i]
			portal = &candidate
			break
		}
	}
	if portal == nil {
		return PortalDetails{}, fmt.Errorf("portal %d: %w", portalID, domain.ErrSimulationInvariant)
	}
	var plane *domain.Plane
	for i := range snapshot.Simulation.Planes {
		if snapshot.Simulation.Planes[i].ID == portal.DestinationPlaneID {
			candidate := snapshot.Simulation.Planes[i]
			plane = &candidate
			break
		}
	}
	if plane == nil {
		return PortalDetails{}, domain.ErrSimulationInvariant
	}
	destination := DestinationDTO{
		PlaneID: portal.DestinationPlaneID, Name: plane.Name, Aliases: append([]string(nil), plane.Aliases...),
		CatalogTier: plane.CatalogTier, Explored: plane.Explored,
	}
	for _, candidate := range snapshot.Simulation.Portals {
		if candidate.DestinationPlaneID == portal.DestinationPlaneID && candidate.ID != portal.ID {
			destination.PreviousConnectionCount++
		}
	}
	for _, observer := range snapshot.Simulation.Observers {
		if observer.CurrentPlaneID == nil || *observer.CurrentPlaneID != portal.DestinationPlaneID {
			continue
		}
		switch observer.Status {
		case domain.ObserverExploring:
			destination.ObserversExploring++
		case domain.ObserverWaitingReturn:
			destination.ObserversWaitingReturn++
		}
	}
	actions, err := quickActions(snapshot.Simulation, *portal, now, cfg)
	if err != nil {
		return PortalDetails{}, err
	}
	details := PortalDetails{
		GeneratedAt: now,
		Portal: PortalViewDTO{
			ID: portal.ID, Name: portal.Name, SlotIndex: portal.SlotIndex, Kind: portal.Kind,
			Status: portal.Status, TerminationReason: portal.TerminationReason,
			Energy: portal.CurrentEnergy(now), Stability: portal.Stability,
			TimeRemainingSeconds: seconds(portal.ScheduledRemaining(now)),
			CreaturesInside:      portal.CreaturesInside(now, cfg), ObserverFlow: portal.ObserverFlow,
			OpenedAt: portal.OpenedAt, ClosedAt: cloneTime(portal.ClosedAt),
			QuickActions: actions,
		},
		Destination: destination,
		History:     BuildEvents(history),
	}
	if risk, ok := portal.RiskLevel(now, cfg); ok {
		details.RiskLevel = &risk
	}
	for _, transit := range transits {
		if transit.PortalID == portalID {
			details.ObserverTransit = &transit
			break
		}
	}
	if recommendation, ok, err := domain.RecommendationForPortal(snapshot.Simulation, portal.ID, now, cfg); err != nil {
		return PortalDetails{}, err
	} else if ok {
		details.Recommendation = &recommendation
	}
	return details, nil
}

// BuildObserverTransits projects authoritative phase timestamps without drawing
// random durations or advancing gameplay. Resolved snapshots normally contain
// only ongoing phases; the countdown itself clamps at the exact deadline.
func BuildObserverTransits(state domain.SimulationState, now time.Time) ([]ObserverTransitDTO, error) {
	portals := make(map[int64]domain.Portal, len(state.Portals))
	for _, portal := range state.Portals {
		if _, exists := portals[portal.ID]; exists || portal.ID <= 0 {
			return nil, domain.ErrSimulationInvariant
		}
		portals[portal.ID] = portal
	}
	seenObservers, busy := make(map[int64]bool), make(map[int64]bool)
	result := make([]ObserverTransitDTO, 0)
	for _, observer := range state.Observers {
		if observer.ID <= 0 || seenObservers[observer.ID] {
			return nil, domain.ErrSimulationInvariant
		}
		seenObservers[observer.ID] = true
		if observer.Status != domain.ObserverOutbound && observer.Status != domain.ObserverReturning {
			continue
		}
		if observer.ActivePortalID == nil || observer.PhaseStartedAt == nil || observer.PhaseEndsAt == nil ||
			!observer.PhaseStartedAt.Before(*observer.PhaseEndsAt) || observer.PhaseStartedAt.After(now) {
			return nil, domain.ErrSimulationInvariant
		}
		portal, exists := portals[*observer.ActivePortalID]
		if !exists || portal.Status != domain.PortalStatusOpen || busy[portal.ID] {
			return nil, domain.ErrSimulationInvariant
		}
		if observer.Status == domain.ObserverOutbound {
			if observer.CurrentPlaneID != nil || portal.ObserverFlow != domain.PortalFlowOutbound {
				return nil, domain.ErrSimulationInvariant
			}
		} else if observer.CurrentPlaneID == nil || *observer.CurrentPlaneID != portal.DestinationPlaneID || portal.ObserverFlow != domain.PortalFlowInbound {
			return nil, domain.ErrSimulationInvariant
		}
		busy[portal.ID] = true
		result = append(result, ObserverTransitDTO{ObserverID: observer.ID, PortalID: portal.ID, Direction: observer.Status,
			StartedAt: observer.PhaseStartedAt.UTC(), CompletesAt: observer.PhaseEndsAt.UTC(), RemainingSeconds: seconds(observer.PhaseEndsAt.Sub(now))})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ObserverID < result[j].ObserverID })
	return result, nil
}

func BuildEvents(events []domain.Event) []EventDTO {
	copyEvents := append([]domain.Event(nil), events...)
	sort.SliceStable(copyEvents, func(i, j int) bool {
		if copyEvents[i].CreatedAt.Equal(copyEvents[j].CreatedAt) {
			return copyEvents[i].ID < copyEvents[j].ID
		}
		return copyEvents[i].CreatedAt.Before(copyEvents[j].CreatedAt)
	})
	result := make([]EventDTO, 0, len(copyEvents))
	for _, event := range copyEvents {
		payload := json.RawMessage(event.PayloadJSON)
		if !json.Valid(payload) {
			payload = json.RawMessage(`{}`)
		}
		result = append(result, EventDTO{
			ID: event.ID, EventType: event.EventType, PortalID: cloneInt64(event.PortalID),
			ObserverID: cloneInt64(event.ObserverID), PlaneID: cloneInt64(event.PlaneID),
			Message: event.Message, PayloadJSON: payload, CreatedAt: event.CreatedAt,
		})
	}
	return result
}

func buildPlane(plane domain.Plane) PlaneDTO {
	return PlaneDTO{ID: plane.ID, Name: plane.Name, Aliases: append([]string(nil), plane.Aliases...), CatalogTier: plane.CatalogTier, Explored: plane.Explored, ExploredAt: cloneTime(plane.ExploredAt)}
}

func buildSlotPortal(state domain.SimulationState, portal domain.Portal, plane domain.Plane, now time.Time, cfg config.Config) (SlotPortalDTO, error) {
	actions, err := quickActions(state, portal, now, cfg)
	if err != nil {
		return SlotPortalDTO{}, err
	}
	return SlotPortalDTO{
		ID: portal.ID, Name: portal.Name, DestinationPlaneID: plane.ID, DestinationPlaneName: plane.Name,
		DestinationExplored: plane.Explored, Energy: portal.CurrentEnergy(now), Stability: portal.Stability,
		TimeRemainingSeconds: seconds(portal.ScheduledRemaining(now)), CreaturesInside: portal.CreaturesInside(now, cfg),
		Status: portal.Status, QuickActions: actions,
	}, nil
}

func quickActions(state domain.SimulationState, portal domain.Portal, now time.Time, cfg config.Config) (QuickActionsDTO, error) {
	var result QuickActionsDTO
	targets := []struct {
		action domain.PortalAction
		set    func(bool, *string)
	}{
		{domain.PortalActionStabilize, func(ok bool, reason *string) { result.CanStabilize, result.StabilizeUnavailableReason = ok, reason }},
		{domain.PortalActionClose, func(ok bool, reason *string) { result.CanClose, result.CloseUnavailableReason = ok, reason }},
		{domain.PortalActionSend, func(ok bool, reason *string) { result.CanSend, result.SendUnavailableReason = ok, reason }},
		{domain.PortalActionRecall, func(ok bool, reason *string) { result.CanRecall, result.RecallUnavailableReason = ok, reason }},
	}
	for _, target := range targets {
		availability, err := domain.PortalActionAvailability(state, portal.ID, target.action, now, cfg)
		if err != nil {
			return QuickActionsDTO{}, err
		}
		if availability.Available {
			target.set(true, nil)
			continue
		}
		descriptor, ok := DescribeDomainError(availability.Cause)
		if !ok || descriptor.Confirmable {
			return QuickActionsDTO{}, domain.ErrSimulationInvariant
		}
		reason := descriptor.Code
		target.set(false, &reason)
	}
	return result, nil
}

func countObserver(counts *ObserverCountsDTO, status domain.ObserverStatus) {
	switch status {
	case domain.ObserverAvailable:
		counts.Available++
		counts.InLab++
	case domain.ObserverOutbound:
		counts.Outbound++
		counts.InTransit++
	case domain.ObserverExploring:
		counts.Exploring++
		counts.InWorlds++
	case domain.ObserverWaitingReturn:
		counts.WaitingReturn++
		counts.InWorlds++
	case domain.ObserverReturning:
		counts.Returning++
		counts.InTransit++
	case domain.ObserverLost:
		counts.Lost++
	}
}

func seconds(duration time.Duration) int64 {
	if duration <= 0 {
		return 0
	}
	return int64(duration / time.Second)
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
