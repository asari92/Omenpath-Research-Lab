package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// RiskLevel (Final Spec §13).
//
// The numeric risk_score is backend-internal; the frontend receives only
// the level. Risk is defined only for OPEN portals.
type RiskLevel string

const (
	RiskLow      RiskLevel = "LOW"      // 0..25
	RiskMedium   RiskLevel = "MEDIUM"   // >25..50
	RiskHigh     RiskLevel = "HIGH"     // >50..75
	RiskCritical RiskLevel = "CRITICAL" // >75..100
)

// RiskScore returns the internal numeric risk (Final Spec §13):
//
//	base = max(0, (safeHorizon − effectiveLifetime) / safeHorizon × 100)
//	risk = min(100, base + 20 if UNSTABLE)
//
// The hidden instability timestamp is deliberately NOT part of the
// formula (RISK-010). The score stays backend-internal: the frontend only
// ever sees the level (RISK-011).
func (p Portal) RiskScore(now time.Time, cfg config.Config) float64 {
	horizon := cfg.RiskSafeHorizon.Seconds()
	effective := p.EffectiveLifetime(now).Seconds()

	base := (horizon - effective) / horizon * 100
	if base < 0 {
		base = 0
	}

	score := base
	if p.Stability == PortalUnstable {
		score += cfg.RiskInstabilityPenalty
	}
	if score > 100 {
		score = 100
	}
	return score
}

// RiskLevel maps the score to the UI-facing band. ok is false for
// terminal portals: risk is defined only while OPEN (resolved decision
// S2-D2) — no level is fabricated for CLOSED/COLLAPSED.
func (p Portal) RiskLevel(now time.Time, cfg config.Config) (RiskLevel, bool) {
	if p.IsTerminal() {
		return "", false
	}
	switch score := p.RiskScore(now, cfg); {
	case score <= 25:
		return RiskLow, true
	case score <= 50:
		return RiskMedium, true
	case score <= 75:
		return RiskHigh, true
	default:
		return RiskCritical, true
	}
}
