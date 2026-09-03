package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// LabState is the laboratory energy baseline (Final Spec §14).
// Current energy is derived: baseline + regen × elapsed, capped at 100;
// it is never written to storage every second (§34).
type LabState struct {
	EnergyBase   int
	EnergyBaseAt time.Time

	// LeylineOverrideUntil is the deadline of the active Leyline Override
	// (Final Spec §22); nil when inactive.
	LeylineOverrideUntil *time.Time
}

// NewLabState creates a laboratory energy baseline at now.
func NewLabState(initialEnergy int, now time.Time, cfg config.Config) (LabState, error) {
	if initialEnergy < 0 || initialEnergy > cfg.LabEnergyMax {
		return LabState{}, ErrLabEnergyInvariant
	}
	return LabState{EnergyBase: initialEnergy, EnergyBaseAt: now}, nil
}

// NewTutorialLabState creates the full-energy baseline required by Tutorial.
func NewTutorialLabState(now time.Time, cfg config.Config) LabState {
	return LabState{EnergyBase: cfg.LabEnergyMax, EnergyBaseAt: now}
}

// CurrentEnergy derives integer energy from the stored baseline without
// mutating it. Only completed whole seconds contribute regeneration.
func (l LabState) CurrentEnergy(now time.Time, cfg config.Config) int {
	energy := l.EnergyBase
	if energy < 0 {
		energy = 0
	}
	if energy > cfg.LabEnergyMax {
		energy = cfg.LabEnergyMax
	}
	if now.Before(l.EnergyBaseAt) {
		return energy
	}

	wholeSeconds := int(now.Sub(l.EnergyBaseAt) / time.Second)
	energy += wholeSeconds * cfg.LabRegenPerSec
	if energy > cfg.LabEnergyMax {
		return cfg.LabEnergyMax
	}
	if energy < 0 {
		return 0
	}
	return energy
}

// AppMode distinguishes the deterministic Tutorial from Live Mode
// (Final Spec §28).
type AppMode string

const (
	ModeTutorial AppMode = "TUTORIAL"
	ModeLive     AppMode = "LIVE"
)

// AppState holds mode-level application state (Final Spec §2, §34).
type AppState struct {
	Mode         AppMode
	TutorialStep int
}
