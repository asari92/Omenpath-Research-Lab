package domain

import (
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// StartOutbound starts an already-authorized Lab-to-Plane transit. Portal
// admission policy is deliberately left to Stage 4; this primitive owns only
// the Observer lifecycle mutation and the one-time duration draw.
func (o *Observer) StartOutbound(now time.Time, portalID int64, rnd random.Random, cfg config.Config) error {
	if o.Status == ObserverLost {
		return ErrObserverLost
	}
	if o.Status != ObserverAvailable {
		return ErrObserverNotAvailable
	}

	durationSeconds := rnd.IntInclusive(
		int(cfg.ObserverTransitMin/time.Second),
		int(cfg.ObserverTransitMax/time.Second),
	)
	startedAt := now
	endsAt := now.Add(time.Duration(durationSeconds) * time.Second)
	activePortalID := portalID

	o.Status = ObserverOutbound
	o.CurrentPlaneID = nil
	o.ActivePortalID = &activePortalID
	o.PhaseStartedAt = &startedAt
	o.PhaseEndsAt = &endsAt
	o.UpdatedAt = now
	return nil
}
