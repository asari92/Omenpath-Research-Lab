package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// MaxCreaturesForTTL returns the maximum initial creature count for a
// portal lifetime (Final Spec §12):
//
//	max(0, min(CreatureMax, floor((ttl − margin) / transit)))
//
// The clearance margin guarantees the last creature finishes passing
// before the portal's scheduled end.
func MaxCreaturesForTTL(ttl time.Duration, cfg config.Config) int {
	maxByTTL := int((ttl - cfg.CreatureClearanceMargin) / cfg.CreatureTransit)
	if maxByTTL < 0 {
		return 0
	}
	if maxByTTL > cfg.CreatureMax {
		return cfg.CreatureMax
	}
	return maxByTTL
}

// CreaturesInside returns the derived current count (Final Spec §12):
// creatures pass one per CreatureTransit seconds, clamped at zero.
// There are no creature death statistics — the corridor simply empties.
func (p Portal) CreaturesInside(now time.Time, cfg config.Config) int {
	elapsed := now.Sub(p.OpenedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	passed := int(elapsed / cfg.CreatureTransit)
	inside := p.CreaturesInitial - passed
	if inside < 0 {
		return 0
	}
	return inside
}
