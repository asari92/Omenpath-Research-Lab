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

// ResolvePortalLifecycleWithLabEmergency atomically couples one Portal
// lifecycle transition to the Lab emergency state. Portal.ClosedAt supplies
// the semantic Collapse time, including during late resolution.
func ResolvePortalLifecycleWithLabEmergency(
	lab *LabState,
	portal *Portal,
	now time.Time,
	cfg config.Config,
) (changed bool, err error) {
	if _, err := prepareLabEnergySpend(lab, now, 0, cfg); err != nil {
		return false, err
	}
	if portal == nil {
		return false, ErrPortalNotOpen
	}

	nextLab := *lab
	nextPortal := *portal
	changed, err = nextPortal.ResolveLifecycle(now)
	if err != nil || !changed {
		return changed, err
	}
	if nextPortal.Status == PortalStatusCollapsed {
		if nextPortal.ClosedAt == nil {
			return false, ErrLabEnergyInvariant
		}
		if err := nextLab.ActivateLeylineOverride(*nextPortal.ClosedAt, cfg); err != nil {
			return false, err
		}
		*lab = nextLab
	}
	*portal = nextPortal
	return true, nil
}
