package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// NeedsAttentionPortalIndex returns the original slice index of the one OPEN
// Portal selected by Final Spec §6. It never sorts or mutates the input.
func NeedsAttentionPortalIndex(portals []Portal, now time.Time, cfg config.Config) (int, bool, error) {
	if cfg.RiskSafeHorizon <= 0 || cfg.RiskInstabilityPenalty < 0 {
		return 0, false, ErrSimulationInvariant
	}
	selected := -1
	for i := range portals {
		if portals[i].Status != PortalStatusOpen {
			continue
		}
		if portals[i].Stability != PortalStable && portals[i].Stability != PortalUnstable {
			return 0, false, ErrSimulationInvariant
		}
		if selected == -1 || attentionBefore(portals[i], portals[selected], now, cfg) {
			selected = i
		}
	}
	if selected == -1 {
		return 0, false, nil
	}
	return selected, true, nil
}

func attentionBefore(a, b Portal, now time.Time, cfg config.Config) bool {
	aScore, bScore := a.RiskScore(now, cfg), b.RiskScore(now, cfg)
	if aScore != bScore {
		return aScore > bScore
	}
	if a.Stability != b.Stability {
		return a.Stability == PortalUnstable
	}
	aLifetime, bLifetime := a.EffectiveLifetime(now), b.EffectiveLifetime(now)
	if aLifetime != bLifetime {
		return aLifetime < bLifetime
	}
	if !a.OpenedAt.Equal(b.OpenedAt) {
		return a.OpenedAt.Before(b.OpenedAt)
	}
	return a.ID < b.ID
}
