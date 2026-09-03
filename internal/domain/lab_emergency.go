package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// LeylineOverrideActive reports whether now is inside the half-open emergency
// window [collapseAt, LeylineOverrideUntil). The start is derived from the
// stored deadline so later energy re-baselines cannot move the window.
func (l LabState) LeylineOverrideActive(now time.Time, cfg config.Config) bool {
	if l.LeylineOverrideUntil == nil || cfg.EmergencyDuration <= 0 {
		return false
	}
	startedAt := l.LeylineOverrideUntil.Add(-cfg.EmergencyDuration)
	return !now.Before(startedAt) && now.Before(*l.LeylineOverrideUntil)
}

// ActivateLeylineOverride applies one Collapse at its semantic event time.
func (l *LabState) ActivateLeylineOverride(collapseAt time.Time, cfg config.Config) error {
	if l == nil || l.EnergyBase < 0 || l.EnergyBase > cfg.LabEnergyMax ||
		collapseAt.Before(l.EnergyBaseAt) || cfg.EmergencyDuration <= 0 {
		return ErrLabEnergyInvariant
	}

	deadline := collapseAt.Add(cfg.EmergencyDuration)
	l.EnergyBase = 0
	l.EnergyBaseAt = collapseAt
	l.LeylineOverrideUntil = &deadline
	return nil
}
