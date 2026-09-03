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

// CanAfford reports whether a valid baseline has enough derived energy at now.
func (l LabState) CanAfford(now time.Time, cost int, cfg config.Config) bool {
	_, err := prepareLabEnergySpend(&l, now, cost, cfg)
	return err == nil
}

// SpendEnergy atomically stores the post-spend energy as a new baseline.
// A zero-cost spend is a true no-op and does not move the baseline timestamp.
func (l *LabState) SpendEnergy(now time.Time, cost int, cfg config.Config) error {
	remaining, err := prepareLabEnergySpend(l, now, cost, cfg)
	if err != nil {
		return err
	}
	commitLabEnergySpend(l, now, cost, remaining)
	return nil
}

func prepareLabEnergySpend(l *LabState, now time.Time, cost int, cfg config.Config) (int, error) {
	if l == nil || l.EnergyBase < 0 || l.EnergyBase > cfg.LabEnergyMax || now.Before(l.EnergyBaseAt) || cost < 0 {
		return 0, ErrLabEnergyInvariant
	}
	current := l.CurrentEnergy(now, cfg)
	if current < cost {
		return 0, ErrInsufficientLabEnergy
	}
	return current - cost, nil
}

func commitLabEnergySpend(l *LabState, now time.Time, cost, remaining int) {
	if cost == 0 {
		return
	}
	l.EnergyBase = remaining
	l.EnergyBaseAt = now
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
