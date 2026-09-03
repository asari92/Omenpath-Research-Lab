package domain

import "time"

// LabState is the laboratory energy baseline (Final Spec §14).
// Current energy is derived: baseline + regen × elapsed, capped at 100;
// it is never written to storage every second (§34).
type LabState struct {
	EnergyBase   int
	EnergyBaseAt time.Time

	// LeylineOverrideUntil is the deadline of the active Leyline Override
	// (Final Spec §22); nil when inactive.
	LeylineOverrideUntil *time.Time
}

// AppMode distinguishes the deterministic Tutorial from Live Mode
// (Final Spec §28).
type AppMode string

const (
	ModeTutorial AppMode = "TUTORIAL"
	ModeLive     AppMode = "LIVE"
)

// AppState holds mode-level application state (Final Spec §2, §34).
type AppState struct {
	Mode         AppMode
	TutorialStep int
}
