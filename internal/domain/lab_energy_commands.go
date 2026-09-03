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
