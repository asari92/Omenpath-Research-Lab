// Package config centralizes tunable balance values (Final Spec §37).
//
// Stage 0 rule: fixed domain semantics live in the domain package;
// only balance numbers live here. Balance may be tuned after simulation,
// but core semantics must not silently change.
package config

import "time"

// Config holds all tunable balance parameters.
type Config struct {
	// Slots & natural spawn.
	MaxActivePortals int
	SpawnDelayMin    time.Duration
	SpawnDelayMax    time.Duration

	// Natural portal lifetime.
	NaturalTTLMin time.Duration
	NaturalTTLMax time.Duration

	// Portal energy.
	PortalEnergyMin float64
	PortalEnergyMax float64
	PortalDecayMin  float64
	PortalDecayMax  float64

	// Stability.
	UnstableProbability    float64
	InstabilityMinLifetime time.Duration // hidden collapse never before opened_at+5s
	InstabilityCloseMargin time.Duration // hidden collapse at least 1s before natural close

	// Creatures.
	CreatureMax             int
	CreatureTransit         time.Duration // one creature passes per 2 sec
	CreatureClearanceMargin time.Duration // TTL safety margin

	// Observers.
	ObserverCount      int
	ObserverTransitMin time.Duration
	ObserverTransitMax time.Duration
	ResearchDuration   time.Duration

	// Laboratory energy.
	LabEnergyMax            int
	LabRegenPerSec          int
	CloseCost               int
	StabilizeCost           int
	StabilizeBoost          float64
	StabilizeMaxStartEnergy float64
	ExtractionCost          int
	ExtractionEnergyMin     float64
	ExtractionEnergyMax     float64
	ExtractionTTLMin        time.Duration
	ExtractionTTLMax        time.Duration
	ExtractionSync          time.Duration

	// Emergency (Leyline Override).
	EmergencyDuration time.Duration

	// Risk.
	RiskSafeHorizon        time.Duration
	RiskInstabilityPenalty float64
}

// Default returns the balance values fixed by Final Spec §37.
func Default() Config {
	return Config{
		MaxActivePortals: 7,
		SpawnDelayMin:    0,
		SpawnDelayMax:    20 * time.Second,

		NaturalTTLMin: 10 * time.Second,
		NaturalTTLMax: 300 * time.Second,

		PortalEnergyMin: 10.0,
		PortalEnergyMax: 100.0,
		PortalDecayMin:  0.1,
		PortalDecayMax:  1.0,

		UnstableProbability:    0.35,
		InstabilityMinLifetime: 5 * time.Second,
		InstabilityCloseMargin: 1 * time.Second,

		CreatureMax:             10,
		CreatureTransit:         2 * time.Second,
		CreatureClearanceMargin: 2 * time.Second,

		ObserverCount:      10,
		ObserverTransitMin: 5 * time.Second,
		ObserverTransitMax: 15 * time.Second,
		ResearchDuration:   20 * time.Second,

		LabEnergyMax:            100,
		LabRegenPerSec:          1,
		CloseCost:               5,
		StabilizeCost:           20,
		StabilizeBoost:          15.0,
		StabilizeMaxStartEnergy: 85.0,
		ExtractionCost:          30,
		ExtractionEnergyMin:     60.0,
		ExtractionEnergyMax:     100.0,
		ExtractionTTLMin:        30 * time.Second,
		ExtractionTTLMax:        60 * time.Second,
		ExtractionSync:          5 * time.Second,

		EmergencyDuration: 20 * time.Second,

		RiskSafeHorizon:        45 * time.Second,
		RiskInstabilityPenalty: 20.0,
	}
}
