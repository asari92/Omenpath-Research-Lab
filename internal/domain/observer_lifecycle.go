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

// StartReturning starts an already-authorized Plane-to-Lab transit. It keeps
// the Plane association until the return succeeds and draws a fresh duration.
func (o *Observer) StartReturning(now time.Time, portalID int64, rnd random.Random, cfg config.Config) error {
	if o.Status == ObserverLost {
		return ErrObserverLost
	}
	if o.Status != ObserverWaitingReturn {
		return ErrObserverNotWaitingReturn
	}
	if o.CurrentPlaneID == nil {
		return ErrObserverInvariant
	}

	durationSeconds := rnd.IntInclusive(
		int(cfg.ObserverTransitMin/time.Second),
		int(cfg.ObserverTransitMax/time.Second),
	)
	startedAt := now
	endsAt := now.Add(time.Duration(durationSeconds) * time.Second)
	activePortalID := portalID

	o.Status = ObserverReturning
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
	case ObserverAvailable, ObserverLost:
		return nil
	case ObserverOutbound:
		return resolveObserverOutbound(o, plane, portal, now, cfg)
	case ObserverExploring:
		return resolveObserverResearch(o, plane, now)
	case ObserverWaitingReturn:
		if o.CurrentPlaneID == nil {
			return ErrObserverInvariant
		}
		return nil
	case ObserverReturning:
		return resolveObserverReturning(o, plane, portal, now)
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
	failed, err := resolveObserverTransitFailure(o, portal, now)
	if err != nil || failed {
		return err
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

func resolveObserverResearch(o *Observer, plane *Plane, now time.Time) error {
	if o.CurrentPlaneID == nil || o.PhaseStartedAt == nil || o.PhaseEndsAt == nil ||
		plane == nil || *o.CurrentPlaneID != plane.ID {
		return ErrObserverInvariant
	}
	if now.Before(*o.PhaseEndsAt) {
		return nil
	}

	completedAt := *o.PhaseEndsAt
	o.Status = ObserverWaitingReturn
	o.ActivePortalID = nil
	o.PhaseStartedAt = &completedAt
	o.PhaseEndsAt = nil
	o.UpdatedAt = completedAt
	return nil
}

func resolveObserverReturning(o *Observer, plane *Plane, portal *Portal, now time.Time) error {
	if o.CurrentPlaneID == nil || o.ActivePortalID == nil || o.PhaseStartedAt == nil ||
		o.PhaseEndsAt == nil || plane == nil || portal == nil ||
		*o.CurrentPlaneID != plane.ID || *o.ActivePortalID != portal.ID ||
		plane.ID != portal.DestinationPlaneID {
		return ErrObserverInvariant
	}
	failed, err := resolveObserverTransitFailure(o, portal, now)
	if err != nil || failed {
		return err
	}
	if now.Before(*o.PhaseEndsAt) {
		return nil
	}

	returnedAt := *o.PhaseEndsAt
	o.Status = ObserverAvailable
	o.CurrentPlaneID = nil
	o.ActivePortalID = nil
	o.PhaseStartedAt = nil
	o.PhaseEndsAt = nil
	o.UpdatedAt = returnedAt

	if !plane.Explored {
		plane.Explored = true
		plane.ExploredAt = &returnedAt
	}
	return nil
}

func resolveObserverTransitFailure(o *Observer, portal *Portal, now time.Time) (bool, error) {
	if portal.Status == PortalStatusOpen {
		return false, nil
	}
	if portal.ClosedAt == nil || o.PhaseEndsAt == nil {
		return false, ErrObserverInvariant
	}
	if !portal.ClosedAt.Before(*o.PhaseEndsAt) || now.Before(*portal.ClosedAt) {
		return false, nil
	}

	lostAt := *portal.ClosedAt
	o.Status = ObserverLost
	o.CurrentPlaneID = nil
	o.ActivePortalID = nil
	o.PhaseStartedAt = nil
	o.PhaseEndsAt = nil
	o.UpdatedAt = lostAt
	return true, nil
}
