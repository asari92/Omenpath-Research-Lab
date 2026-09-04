package domain

import (
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// NaturalSpawnState stores only the current Natural generator schedule.
// A paused generator has nil timestamps; a scheduled generator has both.
type NaturalSpawnState struct {
	ScheduledAt *time.Time
	DueAt       *time.Time
	Paused      bool
}

// SimulationState is the pure active-state aggregate advanced by Stage 8.
// Later LabManager work will provide clock and concurrency ownership.
type SimulationState struct {
	Lab          LabState
	Portals      []Portal
	Planes       []Plane
	Observers    []Observer
	NextPortalID int64
	NaturalSpawn NaturalSpawnState
	LastTickAt   *time.Time
}

type NaturalSpawnResult struct {
	Changed  bool
	Spawned  bool
	PortalID int64
}

type SimulationTickResult struct {
	Changed                bool
	Spawned                bool
	SpawnedPortalID        int64
	HasNeedsAttention      bool
	NeedsAttentionPortalID int64
	LabEnergy              int
}

// NewNaturalSpawnState draws one inclusive whole-second delay from now.
func NewNaturalSpawnState(now time.Time, cfg config.Config, rnd random.Random) (NaturalSpawnState, error) {
	if rnd == nil || !validSpawnDelayConfig(cfg) {
		return NaturalSpawnState{}, ErrSimulationInvariant
	}
	delay := time.Duration(rnd.IntInclusive(
		int(cfg.SpawnDelayMin/time.Second),
		int(cfg.SpawnDelayMax/time.Second),
	)) * time.Second
	scheduledAt := now
	dueAt := now.Add(delay)
	return NaturalSpawnState{ScheduledAt: &scheduledAt, DueAt: &dueAt}, nil
}

func validSpawnDelayConfig(cfg config.Config) bool {
	return cfg.SpawnDelayMin >= 0 && cfg.SpawnDelayMin <= cfg.SpawnDelayMax &&
		cfg.SpawnDelayMin%time.Second == 0 && cfg.SpawnDelayMax%time.Second == 0
}

// ResolveTick initially exposes the approved aggregate validation boundary.
// Ordered lifecycle behavior is added checkpoint-by-checkpoint.
func (state *SimulationState) ResolveTick(now time.Time, _ random.Random, cfg config.Config) (SimulationTickResult, error) {
	if err := validateSimulationState(state, now, cfg); err != nil {
		return SimulationTickResult{}, err
	}
	return SimulationTickResult{LabEnergy: state.Lab.CurrentEnergy(now, cfg)}, nil
}

func validateSimulationState(state *SimulationState, now time.Time, cfg config.Config) error {
	if state == nil || cfg.MaxActivePortals != 7 || cfg.ObserverCount <= 0 ||
		!validSpawnDelayConfig(cfg) || len(state.Planes) != 85 ||
		len(state.Observers) != cfg.ObserverCount || state.NextPortalID <= 0 {
		return ErrSimulationInvariant
	}
	if _, err := prepareLabEnergySpend(&state.Lab, now, 0, cfg); err != nil {
		return ErrSimulationInvariant
	}
	if state.LastTickAt != nil && state.LastTickAt.After(now) {
		return ErrSimulationInvariant
	}
	if !validNaturalSpawnState(state.NaturalSpawn) {
		return ErrSimulationInvariant
	}

	planeIDs := make(map[int64]struct{}, len(state.Planes))
	for _, plane := range state.Planes {
		if plane.ID <= 0 {
			return ErrSimulationInvariant
		}
		if _, duplicate := planeIDs[plane.ID]; duplicate {
			return ErrSimulationInvariant
		}
		planeIDs[plane.ID] = struct{}{}
	}

	portalIDs := make(map[int64]int, len(state.Portals))
	openSlots := make(map[int]struct{}, cfg.MaxActivePortals)
	maxPortalID := int64(0)
	for i := range state.Portals {
		portal := &state.Portals[i]
		if portal.ID <= 0 || !validSimulationPortal(portal, cfg.MaxActivePortals) {
			return ErrSimulationInvariant
		}
		if _, duplicate := portalIDs[portal.ID]; duplicate {
			return ErrSimulationInvariant
		}
		if _, exists := planeIDs[portal.DestinationPlaneID]; !exists {
			return ErrSimulationInvariant
		}
		portalIDs[portal.ID] = i
		if portal.ID > maxPortalID {
			maxPortalID = portal.ID
		}
		if portal.Status == PortalStatusOpen {
			if _, duplicate := openSlots[portal.SlotIndex]; duplicate {
				return ErrSimulationInvariant
			}
			openSlots[portal.SlotIndex] = struct{}{}
		}
	}
	if state.NextPortalID <= maxPortalID {
		return ErrSimulationInvariant
	}
	return validateSimulationObservers(state.Observers, planeIDs, portalIDs, state.Portals)
}

func validNaturalSpawnState(spawn NaturalSpawnState) bool {
	if spawn.Paused {
		return spawn.ScheduledAt == nil && spawn.DueAt == nil
	}
	return spawn.ScheduledAt != nil && spawn.DueAt != nil &&
		!spawn.DueAt.Before(*spawn.ScheduledAt)
}

func validSimulationPortal(portal *Portal, maxSlots int) bool {
	if portal.Kind != PortalKindNatural && portal.Kind != PortalKindExtraction {
		return false
	}
	if portal.Stability != PortalStable && portal.Stability != PortalUnstable {
		return false
	}
	if portal.ObserverFlow != PortalFlowNone && portal.ObserverFlow != PortalFlowOutbound &&
		portal.ObserverFlow != PortalFlowInbound {
		return false
	}
	switch portal.Status {
	case PortalStatusOpen:
		return portal.ClosedAt == nil && portal.TerminationReason == TerminationNone &&
			portal.SlotIndex >= 1 && portal.SlotIndex <= maxSlots
	case PortalStatusClosed:
		return portal.ClosedAt != nil &&
			(portal.TerminationReason == TerminationNaturalClose ||
				portal.TerminationReason == TerminationManualClose)
	case PortalStatusCollapsed:
		return portal.ClosedAt != nil &&
			(portal.TerminationReason == TerminationEnergyDepleted ||
				portal.TerminationReason == TerminationInstability)
	default:
		return false
	}
}

func validateSimulationObservers(
	observers []Observer,
	planeIDs map[int64]struct{},
	portalIDs map[int64]int,
	portals []Portal,
) error {
	ids := make(map[int64]struct{}, len(observers))
	for i := range observers {
		observer := &observers[i]
		if observer.ID <= 0 {
			return ErrSimulationInvariant
		}
		if _, duplicate := ids[observer.ID]; duplicate {
			return ErrSimulationInvariant
		}
		ids[observer.ID] = struct{}{}

		hasPlane := observer.CurrentPlaneID != nil
		hasPortal := observer.ActivePortalID != nil
		hasStart := observer.PhaseStartedAt != nil
		hasEnd := observer.PhaseEndsAt != nil
		if hasPlane {
			if _, exists := planeIDs[*observer.CurrentPlaneID]; !exists {
				return ErrSimulationInvariant
			}
		}
		if hasPortal {
			index, exists := portalIDs[*observer.ActivePortalID]
			if !exists {
				return ErrSimulationInvariant
			}
			if hasPlane && portals[index].DestinationPlaneID != *observer.CurrentPlaneID {
				return ErrSimulationInvariant
			}
		}

		switch observer.Status {
		case ObserverAvailable, ObserverLost:
			if hasPlane || hasPortal || hasStart || hasEnd {
				return ErrSimulationInvariant
			}
		case ObserverOutbound:
			if hasPlane || !hasPortal || !validSimulationPhase(observer) {
				return ErrSimulationInvariant
			}
		case ObserverExploring:
			if !hasPlane || hasPortal || !validSimulationPhase(observer) {
				return ErrSimulationInvariant
			}
		case ObserverWaitingReturn:
			if !hasPlane || hasPortal || !hasStart || hasEnd {
				return ErrSimulationInvariant
			}
		case ObserverReturning:
			if !hasPlane || !hasPortal || !validSimulationPhase(observer) {
				return ErrSimulationInvariant
			}
		default:
			return ErrSimulationInvariant
		}
	}
	return nil
}

func validSimulationPhase(observer *Observer) bool {
	return observer.PhaseStartedAt != nil && observer.PhaseEndsAt != nil &&
		observer.PhaseStartedAt.Before(*observer.PhaseEndsAt)
}
