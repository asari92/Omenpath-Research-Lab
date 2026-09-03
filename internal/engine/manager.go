// Package engine will host LabManager — the in-memory active state that
// orchestrates REST commands, the 1-second simulation loop and the
// WebSocket hub behind a mutex (Final Spec §31, §32).
//
// Stage 1: package layout placeholder only. Behavior arrives in later
// stages per 02_IMPLEMENTATION_ROADMAP_TDD.md.
package engine

import (
	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
)

// LabManager owns the active lab state. No behavior implemented yet.
type LabManager struct {
	cfg   config.Config
	clock clock.Clock
}

// NewLabManager wires the manager with balance config and time source.
func NewLabManager(cfg config.Config, clk clock.Clock) *LabManager {
	return &LabManager{cfg: cfg, clock: clk}
}
