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

// ResolveObserverLifecycle advances deterministic Observer phases to the
// state effective at now. Later checkpoints extend this resolver with
// research completion, return, transit failure, and multi-phase catch-up.
func ResolveObserverLifecycle(o *Observer, plane *Plane, portal *Portal, now time.Time, cfg config.Config) error {
	switch o.Status {
	case ObserverAvailable, ObserverWaitingReturn, ObserverLost:
		return nil
	case ObserverOutbound:
		return resolveObserverOutbound(o, plane, portal, now, cfg)
	default:
		return ErrObserverInvariant
	}
}

func resolveObserverOutbound(o *Observer, plane *Plane, portal *Portal, now time.Time, cfg config.Config) error {
	if o.ActivePortalID == nil || o.PhaseStartedAt == nil || o.PhaseEndsAt == nil ||
		plane == nil || portal == nil || *o.ActivePortalID != portal.ID ||
		plane.ID != portal.DestinationPlaneID {
		return ErrObserverInvariant
	}
	if now.Before(*o.PhaseEndsAt) {
		return nil
	}

	arrivedAt := *o.PhaseEndsAt
	planeID := plane.ID
	researchEndsAt := arrivedAt.Add(cfg.ResearchDuration)
	o.Status = ObserverExploring
	o.CurrentPlaneID = &planeID
	o.ActivePortalID = nil
	o.PhaseStartedAt = &arrivedAt
	o.PhaseEndsAt = &researchEndsAt
	o.UpdatedAt = arrivedAt
	return nil
}
