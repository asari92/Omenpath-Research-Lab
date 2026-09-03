package domain

import (
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// SendObserverWithLabEnergy applies the zero-cost SEND contract and delegates
// all Portal/Observer rules to the completed Stage 4 command.
func SendObserverWithLabEnergy(
	lab *LabState,
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	if _, err := prepareLabEnergySpend(lab, now, 0, cfg); err != nil {
		return 0, err
	}
	return SendObserver(portal, plane, observers, now, confirmUnstable, rnd, cfg)
}

// RecallObserverWithLabEnergy applies the zero-cost RECALL contract and
// delegates all Portal/Observer rules to the completed Stage 4 command.
func RecallObserverWithLabEnergy(
	lab *LabState,
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirmUnstable bool,
	rnd random.Random,
	cfg config.Config,
) (observerID int64, err error) {
	if _, err := prepareLabEnergySpend(lab, now, 0, cfg); err != nil {
		return 0, err
	}
	return RecallObserver(portal, plane, observers, now, confirmUnstable, rnd, cfg)
}

// ClosePortalWithLabEnergy preflights the configured ordinary Close cost,
// delegates Observer-aware closure, and commits one debit only on success.
func ClosePortalWithLabEnergy(
	lab *LabState,
	portal *Portal,
	plane *Plane,
	observers []Observer,
	now time.Time,
	confirm bool,
	cfg config.Config,
) error {
	cost := effectiveCloseCost(lab, now, cfg)
	remaining, err := prepareLabEnergySpend(lab, now, cost, cfg)
	if err != nil {
		return err
	}
	if err := ClosePortalWithObservers(portal, plane, observers, now, confirm, cfg); err != nil {
		return err
	}
	commitLabEnergySpend(lab, now, cost, remaining)
	return nil
}

func effectiveCloseCost(lab *LabState, now time.Time, cfg config.Config) int {
	if lab != nil && lab.LeylineOverrideActive(now, cfg) {
		return 0
	}
	return cfg.CloseCost
}

// StabilizePortalWithLabEnergy preflights the configured ordinary Stabilize
// cost and commits one debit only after Portal.Stabilize succeeds.
func StabilizePortalWithLabEnergy(
	lab *LabState,
	portal *Portal,
	now time.Time,
	cfg config.Config,
) error {
	remaining, err := prepareLabEnergySpend(lab, now, cfg.StabilizeCost, cfg)
	if err != nil {
		return err
	}
	if err := portal.Stabilize(now, cfg); err != nil {
		return err
	}
	commitLabEnergySpend(lab, now, cfg.StabilizeCost, remaining)
	return nil
}
