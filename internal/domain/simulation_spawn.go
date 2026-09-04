package domain

import (
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// ResolveNaturalSpawn atomically advances only the Natural generator stage.
// The full tick reuses the prepared helper after aggregate preflight.
func ResolveNaturalSpawn(
	state *SimulationState,
	now time.Time,
	rnd random.Random,
	cfg config.Config,
) (NaturalSpawnResult, error) {
	if err := validateSimulationState(state, now, cfg); err != nil {
		return NaturalSpawnResult{}, err
	}
	next := cloneSimulationState(*state)
	result, err := resolveNaturalSpawnPrepared(&next, now, rnd, cfg)
	if err != nil {
		return NaturalSpawnResult{}, err
	}
	*state = next
	return result, nil
}

func resolveNaturalSpawnPrepared(
	state *SimulationState,
	now time.Time,
	rnd random.Random,
	cfg config.Config,
) (NaturalSpawnResult, error) {
	openCount := countOpenPortals(state.Portals)
	if openCount >= cfg.MaxActivePortals {
		changed := !state.NaturalSpawn.Paused || state.NaturalSpawn.ScheduledAt != nil || state.NaturalSpawn.DueAt != nil
		state.NaturalSpawn = NaturalSpawnState{Paused: true}
		return NaturalSpawnResult{Changed: changed}, nil
	}
	if state.NaturalSpawn.Paused {
		next, err := NewNaturalSpawnState(now, cfg, rnd)
		if err != nil {
			return NaturalSpawnResult{}, err
		}
		state.NaturalSpawn = next
		return NaturalSpawnResult{Changed: true}, nil
	}
	if !naturalSpawnDue(state.NaturalSpawn, now) {
		return NaturalSpawnResult{}, nil
	}

	// Checkpoint D adds the due/free spawn branch.
	return NaturalSpawnResult{}, nil
}

func countOpenPortals(portals []Portal) int {
	count := 0
	for i := range portals {
		if portals[i].Status == PortalStatusOpen {
			count++
		}
	}
	return count
}

func naturalSpawnDue(spawn NaturalSpawnState, now time.Time) bool {
	return spawn.ScheduledAt != nil && spawn.DueAt != nil &&
		now.After(*spawn.ScheduledAt) && !now.Before(*spawn.DueAt)
}
