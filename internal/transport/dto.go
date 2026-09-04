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
	Mode         domain.AppMode `json:"mode"`
	TutorialStep int            `json:"tutorial_step"`
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

type PortalCountsDTO struct {
	Active    int `json:"active"`
	Maximum   int `json:"maximum"`
	Critical  int `json:"critical"`
	Closed    int `json:"closed"`
	Collapsed int `json:"collapsed"`
}

type PlaneDTO struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Aliases     []string   `json:"aliases"`
	CatalogTier string     `json:"catalog_tier"`
	Explored    bool       `json:"explored"`
	ExploredAt  *time.Time `json:"explored_at"`
}

type QuickActionsDTO struct {
	CanStabilize bool `json:"can_stabilize"`
	CanClose     bool `json:"can_close"`
	CanSend      bool `json:"can_send_observer"`
	CanRecall    bool `json:"can_recall_observer"`
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
	GeneratedAt            time.Time         `json:"generated_at"`
	App                    AppDTO            `json:"app"`
	Lab                    LabDTO            `json:"lab"`
	Exploration            ExplorationDTO    `json:"exploration"`
	Observers              ObserverCountsDTO `json:"observers"`
	Portals                PortalCountsDTO   `json:"portals"`
	NeedsAttentionPortalID *int64            `json:"needs_attention_portal_id"`
	Slots                  []SlotDTO         `json:"slots"`
	Planes                 []PlaneDTO        `json:"planes"`
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
	GeneratedAt time.Time         `json:"generated_at"`
	Portal      PortalViewDTO     `json:"portal"`
	Destination DestinationDTO    `json:"destination"`
	RiskLevel   *domain.RiskLevel `json:"risk_level"`
	History     []EventDTO        `json:"history"`
}

func BuildStateSnapshot(snapshot persistence.Snapshot, now time.Time, cfg config.Config) (StateSnapshot, error) {
	now = now.UTC()
	planes := make(map[int64]domain.Plane, len(snapshot.Simulation.Planes))
	result := StateSnapshot{
		GeneratedAt: now,
		App:         AppDTO{Mode: snapshot.App.Mode, TutorialStep: snapshot.App.TutorialStep},
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
	for i, plane := range snapshot.Simulation.Planes {
		if plane.ID <= 0 {
			return StateSnapshot{}, domain.ErrSimulationInvariant
		}
		planes[plane.ID] = plane
		if plane.Explored {
			result.Exploration.Explored++
		}
		result.Planes = append(result.Planes, buildPlane(plane))
		_ = i
	}
	for _, observer := range snapshot.Simulation.Observers {
		countObserver(&result.Observers, observer.Status)
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
			view := buildSlotPortal(snapshot.Simulation, portal, plane, now, cfg)
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
	details := PortalDetails{
		GeneratedAt: now,
		Portal: PortalViewDTO{
			ID: portal.ID, Name: portal.Name, SlotIndex: portal.SlotIndex, Kind: portal.Kind,
			Status: portal.Status, TerminationReason: portal.TerminationReason,
			Energy: portal.CurrentEnergy(now), Stability: portal.Stability,
			TimeRemainingSeconds: seconds(portal.ScheduledRemaining(now)),
			CreaturesInside:      portal.CreaturesInside(now, cfg), ObserverFlow: portal.ObserverFlow,
			OpenedAt: portal.OpenedAt, ClosedAt: cloneTime(portal.ClosedAt),
			QuickActions: quickActions(snapshot.Simulation, *portal, now, cfg),
		},
		Destination: destination,
		History:     BuildEvents(history),
	}
	if risk, ok := portal.RiskLevel(now, cfg); ok {
		details.RiskLevel = &risk
	}
	return details, nil
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

func buildSlotPortal(state domain.SimulationState, portal domain.Portal, plane domain.Plane, now time.Time, cfg config.Config) SlotPortalDTO {
	return SlotPortalDTO{
		ID: portal.ID, Name: portal.Name, DestinationPlaneID: plane.ID, DestinationPlaneName: plane.Name,
		DestinationExplored: plane.Explored, Energy: portal.CurrentEnergy(now), Stability: portal.Stability,
		TimeRemainingSeconds: seconds(portal.ScheduledRemaining(now)), CreaturesInside: portal.CreaturesInside(now, cfg),
		Status: portal.Status, QuickActions: quickActions(state, portal, now, cfg),
	}
}

func quickActions(state domain.SimulationState, portal domain.Portal, now time.Time, cfg config.Config) QuickActionsDTO {
	if portal.Status != domain.PortalStatusOpen {
		return QuickActionsDTO{}
	}
	lab := state.Lab
	closeCost := cfg.CloseCost
	stabilizeCost := cfg.StabilizeCost
	if lab.LeylineOverrideActive(now, cfg) {
		closeCost, stabilizeCost = 0, 0
	}
	actions := QuickActionsDTO{
		CanClose: lab.CanAfford(now, closeCost, cfg),
		CanStabilize: portal.Stability == domain.PortalUnstable && portal.CurrentEnergy(now) <= cfg.StabilizeMaxStartEnergy &&
			lab.CanAfford(now, stabilizeCost, cfg),
	}
	risk, _ := portal.RiskLevel(now, cfg)
	if risk == domain.RiskCritical || portal.CreaturesInside(now, cfg) > 0 {
		return actions
	}
	if _, busy, err := domain.ActiveTransitObserverIndex(state.Observers, portal.ID, now); err != nil || busy {
		return actions
	}
	_, available := domain.AvailableObserverIndex(state.Observers)
	actions.CanSend = portal.ObserverFlow != domain.PortalFlowInbound && available
	_, waiting, err := domain.LongestWaitingObserverIndex(state.Observers, portal.DestinationPlaneID)
	actions.CanRecall = err == nil && waiting && portal.ObserverFlow != domain.PortalFlowOutbound &&
		(portal.Kind != domain.PortalKindExtraction || portal.ExtractionSynchronizedAt != nil)
	return actions
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
