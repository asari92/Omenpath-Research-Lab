package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

// ClosePortalWithObservers performs the Observer-aware part of manual Close.
// Laboratory Energy accounting remains outside this Stage 4 domain command.
func ClosePortalWithObservers(
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirm bool,
	cfg config.Config,
) error {
	if portal.Status != PortalStatusOpen {
		return ErrPortalNotOpen
	}

	index, active, err := ActiveTransitObserverIndex(observers, portal.ID, now)
	if err != nil {
		return err
	}
	if active && !confirm {
		return ErrConfirmationRequired
	}
	if err := portal.Close(now, confirm, cfg); err != nil {
		return err
	}
	if !active {
		return nil
	}
	return ResolveObserverLifecycle(&observers[index], plane, portal, now, cfg)
}
