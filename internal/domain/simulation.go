package domain

import (
	"sort"
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

func cloneSimulationState(state SimulationState) SimulationState {
	state.Portals = append([]Portal(nil), state.Portals...)
	state.Planes = append([]Plane(nil), state.Planes...)
	state.Observers = append([]Observer(nil), state.Observers...)
	return state
}

// ResolveTick atomically advances the pure simulation aggregate in the
// domain-defined stage order. Later checkpoints fill the observer and
// extraction stages; the ordering boundary is established here.
func (state *SimulationState) ResolveTick(now time.Time, rnd random.Random, cfg config.Config) (SimulationTickResult, error) {
	if err := validateSimulationState(state, now, cfg); err != nil {
		return SimulationTickResult{}, err
	}
	if state.LastTickAt != nil && state.LastTickAt.Equal(now) {
		return deriveSimulationTickResult(*state, now, cfg), nil
	}

	next := cloneSimulationState(*state)
	if err := resolvePortalStage(&next, now, cfg); err != nil {
		return SimulationTickResult{}, err
	}
	if err := resolveObserverStage(&next, now, cfg); err != nil {
		return SimulationTickResult{}, err
	}
	if err := resolveExtractionStage(&next, now, rnd, cfg); err != nil {
		return SimulationTickResult{}, err
	}
	spawn, err := resolveNaturalSpawnPrepared(&next, now, rnd, cfg)
	if err != nil {
		return SimulationTickResult{}, err
	}

	tickAt := now
	next.LastTickAt = &tickAt
	result := deriveSimulationTickResult(next, now, cfg)
	result.Changed = true
	result.Spawned = spawn.Spawned
	result.SpawnedPortalID = spawn.PortalID
	*state = next
	return result, nil
}

type portalLifecycleTransition struct {
	index    int
	portalID int64
	closedAt time.Time
}

func resolvePortalStage(state *SimulationState, now time.Time, cfg config.Config) error {
	transitions := make([]portalLifecycleTransition, 0, len(state.Portals))
	for i := range state.Portals {
		candidate := state.Portals[i]
		changed, err := candidate.ResolveLifecycle(now)
		if err != nil {
			return err
		}
		if !changed {
			continue
		}
		if candidate.ClosedAt == nil {
			return ErrSimulationInvariant
		}
		transitions = append(transitions, portalLifecycleTransition{
			index:    i,
			portalID: candidate.ID,
			closedAt: *candidate.ClosedAt,
		})
	}

	sort.Slice(transitions, func(i, j int) bool {
		if transitions[i].closedAt.Equal(transitions[j].closedAt) {
			return transitions[i].portalID < transitions[j].portalID
		}
		return transitions[i].closedAt.Before(transitions[j].closedAt)
	})
	for _, transition := range transitions {
		if _, err := ResolvePortalLifecycleWithLabEmergency(
			&state.Lab,
			&state.Portals[transition.index],
			now,
			cfg,
		); err != nil {
			return err
		}
	}
	return nil
}

func resolveObserverStage(state *SimulationState, now time.Time, cfg config.Config) error {
	planeIndexByID := make(map[int64]int, len(state.Planes))
	initiallyUnexplored := make(map[int64]bool, len(state.Planes))
	for i := range state.Planes {
		planeIndexByID[state.Planes[i].ID] = i
		initiallyUnexplored[state.Planes[i].ID] = !state.Planes[i].Explored
	}
	earliestReturnByPlane := make(map[int64]time.Time)
	portalIndexByID := make(map[int64]int, len(state.Portals))
	for i := range state.Portals {
		portalIndexByID[state.Portals[i].ID] = i
	}

	indices := make([]int, len(state.Observers))
	for i := range indices {
		indices[i] = i
	}
	sort.Slice(indices, func(i, j int) bool {
		return state.Observers[indices[i]].ID < state.Observers[indices[j]].ID
	})

	for _, index := range indices {
		observer := &state.Observers[index]
		var plane *Plane
		var portal *Portal
		beforeStatus := observer.Status
		var returnPlaneID int64
		var returnedAt time.Time
		hasReturnCandidate := beforeStatus == ObserverReturning &&
			observer.CurrentPlaneID != nil && observer.PhaseEndsAt != nil
		if hasReturnCandidate {
			returnPlaneID = *observer.CurrentPlaneID
			returnedAt = *observer.PhaseEndsAt
		}

		if observer.CurrentPlaneID != nil {
			planeIndex, ok := planeIndexByID[*observer.CurrentPlaneID]
			if !ok {
				return ErrSimulationInvariant
			}
			plane = &state.Planes[planeIndex]
		}
		if observer.ActivePortalID != nil {
			portalIndex, ok := portalIndexByID[*observer.ActivePortalID]
			if !ok {
				return ErrSimulationInvariant
			}
			portal = &state.Portals[portalIndex]
			if plane == nil {
				planeIndex, ok := planeIndexByID[portal.DestinationPlaneID]
				if !ok {
					return ErrSimulationInvariant
				}
				plane = &state.Planes[planeIndex]
			}
		}

		if err := ResolveObserverLifecycle(observer, plane, portal, now, cfg); err != nil {
			return err
		}
		if hasReturnCandidate && observer.Status == ObserverAvailable && initiallyUnexplored[returnPlaneID] {
			current, exists := earliestReturnByPlane[returnPlaneID]
			if !exists || returnedAt.Before(current) {
				earliestReturnByPlane[returnPlaneID] = returnedAt
			}
		}
	}
	for planeID, exploredAt := range earliestReturnByPlane {
		index := planeIndexByID[planeID]
		state.Planes[index].Explored = true
		at := exploredAt
		state.Planes[index].ExploredAt = &at
	}
	return nil
}

func resolveExtractionStage(state *SimulationState, now time.Time, rnd random.Random, cfg config.Config) error {
	planeIndexByID := make(map[int64]int, len(state.Planes))
	for i := range state.Planes {
		planeIndexByID[state.Planes[i].ID] = i
	}
	indices := make([]int, 0)
	for i := range state.Portals {
		if state.Portals[i].Kind == PortalKindExtraction {
			indices = append(indices, i)
		}
	}
	sort.Slice(indices, func(i, j int) bool {
		return state.Portals[indices[i]].ID < state.Portals[indices[j]].ID
	})

	for _, index := range indices {
		portal := &state.Portals[index]
		planeIndex, ok := planeIndexByID[portal.DestinationPlaneID]
		if !ok {
			return ErrSimulationInvariant
		}
		if _, _, err := ResolveExtractionSynchronization(
			portal,
			&state.Planes[planeIndex],
			state.Observers,
			now,
			rnd,
			cfg,
		); err != nil {
			return err
		}
	}
	return nil
}

func deriveSimulationTickResult(state SimulationState, now time.Time, cfg config.Config) SimulationTickResult {
	result := SimulationTickResult{LabEnergy: state.Lab.CurrentEnergy(now, cfg)}
	index, ok, err := NeedsAttentionPortalIndex(state.Portals, now, cfg)
	if err == nil && ok {
		result.HasNeedsAttention = true
		result.NeedsAttentionPortalID = state.Portals[index].ID
	}
	return result
}

func validateSimulationState(state *SimulationState, now time.Time, cfg config.Config) error {
	if state == nil || !validSimulationConfig(cfg) || len(state.Planes) != 85 ||
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

func validSimulationConfig(cfg config.Config) bool {
	return cfg.MaxActivePortals == 7 &&
		validSpawnDelayConfig(cfg) &&
		cfg.NaturalTTLMin > 0 && cfg.NaturalTTLMin <= cfg.NaturalTTLMax &&
		cfg.NaturalTTLMin%time.Second == 0 && cfg.NaturalTTLMax%time.Second == 0 &&
		cfg.PortalEnergyMin >= 0 && cfg.PortalEnergyMin <= cfg.PortalEnergyMax &&
		cfg.PortalDecayMin > 0 && cfg.PortalDecayMin <= cfg.PortalDecayMax &&
		cfg.UnstableProbability >= 0 && cfg.UnstableProbability <= 1 &&
		cfg.InstabilityMinLifetime > 0 && cfg.InstabilityCloseMargin > 0 &&
		cfg.NaturalTTLMin >= cfg.InstabilityMinLifetime+cfg.InstabilityCloseMargin &&
		cfg.CreatureMax >= 0 && cfg.CreatureTransit > 0 && cfg.CreatureClearanceMargin >= 0 &&
		cfg.ObserverCount > 0 && cfg.ObserverTransitMin > 0 &&
		cfg.ObserverTransitMin <= cfg.ObserverTransitMax &&
		cfg.ObserverTransitMin%time.Second == 0 && cfg.ObserverTransitMax%time.Second == 0 &&
		cfg.ResearchDuration > 0 &&
		cfg.LabEnergyMax > 0 && cfg.LabRegenPerSec >= 0 &&
		cfg.CloseCost >= 0 && cfg.StabilizeCost >= 0 && cfg.ExtractionCost >= 0 &&
		cfg.StabilizeBoost >= 0 && cfg.StabilizeMaxStartEnergy >= 0 &&
		validExtractionConfig(cfg) &&
		cfg.ExtractionTTLMin%time.Second == 0 && cfg.ExtractionTTLMax%time.Second == 0 &&
		cfg.EmergencyDuration > 0 &&
		cfg.RiskSafeHorizon > 0 && cfg.RiskInstabilityPenalty >= 0
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
