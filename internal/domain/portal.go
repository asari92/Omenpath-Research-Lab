package domain

import (
	"math"
	"time"

	"omenpath-lab/internal/config"
)

// PortalKind (Final Spec §4).
type PortalKind string

const (
	PortalKindNatural    PortalKind = "NATURAL"
	PortalKindExtraction PortalKind = "EXTRACTION"
)

// PortalStatus (Final Spec §4). CLOSED and COLLAPSED are different outcomes.
type PortalStatus string

const (
	PortalStatusOpen      PortalStatus = "OPEN"
	PortalStatusClosed    PortalStatus = "CLOSED"
	PortalStatusCollapsed PortalStatus = "COLLAPSED"
)

// PortalStability (Final Spec §10).
type PortalStability string

const (
	PortalStable   PortalStability = "STABLE"
	PortalUnstable PortalStability = "UNSTABLE"
)

// PortalFlow is the one-way observer direction fixed by first use
// (Final Spec §17).
type PortalFlow string

const (
	PortalFlowNone     PortalFlow = "NONE"
	PortalFlowOutbound PortalFlow = "OUTBOUND" // Lab → Plane only
	PortalFlowInbound  PortalFlow = "INBOUND"  // Plane → Lab only
)

// TerminationReason (Final Spec §4).
type TerminationReason string

const (
	TerminationNone           TerminationReason = ""
	TerminationNaturalClose   TerminationReason = "NATURAL_CLOSE"
	TerminationManualClose    TerminationReason = "MANUAL_CLOSE"
	TerminationEnergyDepleted TerminationReason = "ENERGY_DEPLETED"
	TerminationInstability    TerminationReason = "INSTABILITY"
)

// Portal is a unique instance of a single opening (Final Spec §4):
// a new connection to the same Plane is a new Portal.
//
// Derived realtime values — current energy, creatures inside, risk,
// remaining time — are computed from baselines and are never stored here
// (Final Spec §9, §12, §13, §34).
type Portal struct {
	ID                 int64
	Name               string
	SlotIndex          int
	Kind               PortalKind
	DestinationPlaneID int64

	EnergyBase      float64   // %, baseline at EnergyBaseAt
	EnergyBaseAt    time.Time // when the baseline was set
	EnergyDecayRate float64   // %/sec, hidden from the user

	Stability             PortalStability
	InstabilityCollapseAt *time.Time // hidden; only for UNSTABLE (§10)

	OpenedAt         time.Time
	ScheduledCloseAt time.Time // natural TTL target (§8)

	CreaturesInitial int // count at open; current count is derived (§12)

	ObserverFlow PortalFlow

	// ExtractionSynchronizedAt records the one completed synchronization
	// attempt for an EXTRACTION portal. Natural portals keep it nil.
	ExtractionSynchronizedAt *time.Time

	Status            PortalStatus
	TerminationReason TerminationReason
	ClosedAt          *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsTerminal reports whether the portal has reached a terminal status.
// CLOSED and COLLAPSED never return to OPEN (PORTAL-009).
func (p Portal) IsTerminal() bool {
	return p.Status != PortalStatusOpen
}

// ScheduledRemaining returns the time left until natural close,
// clamped at zero (Final Spec §8). It is derived, never stored.
// Terminal portals have no countdown left: they report 0 even when the
// scheduled close lies in the future at query time (PORTAL-009 extended
// to derived state).
func (p Portal) ScheduledRemaining(now time.Time) time.Duration {
	if p.IsTerminal() {
		return 0
	}
	remaining := p.ScheduledCloseAt.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CurrentEnergy returns the derived realtime energy
// max(0, energy_base − elapsed × decay) (Final Spec §9).
// No per-second storage happens anywhere: callers compute on demand.
// Terminal portals stop behaving like active ones: the energy freezes at
// the ClosedAt moment and never decays further (PORTAL-009 extended to
// derived state). Valid transitions always set ClosedAt; a terminal portal
// without it (hand-built fixture) falls back to live derivation.
func (p Portal) CurrentEnergy(now time.Time) float64 {
	if p.IsTerminal() && p.ClosedAt != nil {
		now = *p.ClosedAt
	}
	elapsed := now.Sub(p.EnergyBaseAt)
	if elapsed < 0 {
		elapsed = 0
	}
	energy := p.EnergyBase - elapsed.Seconds()*p.EnergyDecayRate
	if energy < 0 {
		return 0
	}
	return energy
}

// EnergyDepletionAt returns the semantic moment the energy reaches zero:
// EnergyBaseAt + EnergyBase/DecayRate (Final Spec §9).
// ok is false when the energy can never deplete (no positive decay).
func (p Portal) EnergyDepletionAt() (at time.Time, ok bool) {
	if p.EnergyDecayRate <= 0 {
		return time.Time{}, false
	}
	seconds := p.EnergyBase / p.EnergyDecayRate
	return p.EnergyBaseAt.Add(time.Duration(seconds * float64(time.Second))), true
}

// EnergyLifetime converts the current energy into seconds of remaining
// life at the current decay rate (Final Spec §13).
// With no positive decay the lifetime is effectively infinite.
func (p Portal) EnergyLifetime(now time.Time) time.Duration {
	if p.EnergyDecayRate <= 0 {
		return time.Duration(math.MaxInt64) // ~292 years; safe inside min()
	}
	seconds := p.CurrentEnergy(now) / p.EnergyDecayRate
	return time.Duration(seconds * float64(time.Second))
}

// EffectiveLifetime returns min(scheduled remaining, energy lifetime)
// (Final Spec §13) — the portal's realistic remaining usefulness.
func (p Portal) EffectiveLifetime(now time.Time) time.Duration {
	remaining := p.ScheduledRemaining(now)
	lifetime := p.EnergyLifetime(now)
	if lifetime < remaining {
		return lifetime
	}
	return remaining
}

// ResolveLifecycle advances an OPEN portal to its terminal state when a
// scheduled termination moment has passed (Final Spec §8 natural close,
// §9 energy collapse).
//
// The earliest applicable moment wins; an exact tie between natural close
// and energy depletion resolves to NATURAL_CLOSE so behavior never depends
// on tick ordering. ClosedAt stores the semantic event time, not the time
// the resolution happened to run.
//
// Terminal portals are left untouched (PORTAL-009).
//
// Candidates are considered by earliest semantic moment (Stage 2 plan
// Rule D) with Natural Close winning exact ties (Rules B and C). An exact
// tie between energy depletion and the hidden instability moment keeps
// ENERGY_DEPLETED — a fixed, documented choice: such a tie is reachable
// from the valid factory (both moments may lie strictly inside the TTL
// window), so the ordering must stay deterministic; the regression is
// locked by TestPortal_EnergyDepletionWinsInstabilityTie.
func (p *Portal) ResolveLifecycle(now time.Time) (changed bool, err error) {
	if p.IsTerminal() {
		return false, nil
	}

	deadline := p.ScheduledCloseAt
	reason := TerminationNaturalClose
	status := PortalStatusClosed

	if depletion, ok := p.EnergyDepletionAt(); ok && depletion.Before(deadline) {
		deadline = depletion
		reason = TerminationEnergyDepleted
		status = PortalStatusCollapsed
	}

	// Hidden instability collapse (Final Spec §10): only while the portal
	// is still OPEN and UNSTABLE. Stabilize clears the timestamp, so a
	// stabilized portal can never hit this branch.
	if p.Stability == PortalUnstable && p.InstabilityCollapseAt != nil &&
		p.InstabilityCollapseAt.Before(deadline) {
		deadline = *p.InstabilityCollapseAt
		reason = TerminationInstability
		status = PortalStatusCollapsed
	}

	if now.Before(deadline) {
		return false, nil
	}

	p.Status = status
	p.TerminationReason = reason
	closedAt := deadline
	p.ClosedAt = &closedAt
	p.UpdatedAt = now
	return true, nil
}

// Stabilize is the portal-level primitive (Final Spec §11):
//
//	OPEN + UNSTABLE + current energy ≤ 85%
//	→ STABLE, hidden collapse cleared, current energy +15 as new baseline.
//
// Laboratory Energy cost (20, or 0 during Leyline Override), events and
// broadcasts are orchestrated in later stages — Stage 2 must not couple
// Portal to LabState (stage boundary, plan §3).
func (p *Portal) Stabilize(now time.Time, cfg config.Config) error {
	if p.IsTerminal() {
		return ErrPortalNotOpen
	}
	if p.Stability == PortalStable {
		return ErrPortalAlreadyStable
	}
	if current := p.CurrentEnergy(now); current > cfg.StabilizeMaxStartEnergy {
		return ErrPortalOverchargeRisk
	}

	p.Stability = PortalStable
	p.InstabilityCollapseAt = nil
	p.EnergyBase = p.CurrentEnergy(now) + cfg.StabilizeBoost
	p.EnergyBaseAt = now
	p.UpdatedAt = now
	return nil
}

// Close is the manual-close primitive (Final Spec §21): portal-level state
// transition plus creature-confirmation semantics.
//
// Laboratory Energy cost (5, or 0 during Leyline Override) and the
// observer-in-transit confirmation are orchestrated in later stages —
// Stage 2 must not couple Portal to LabState or Observers (plan §3).
// For a manual action the semantic event time is `now` itself.
func (p *Portal) Close(now time.Time, confirmCreatureInterrupt bool, cfg config.Config) error {
	if p.IsTerminal() {
		return ErrPortalNotOpen
	}
	if p.CreaturesInside(now, cfg) > 0 && !confirmCreatureInterrupt {
		return ErrConfirmationRequired
	}

	p.Status = PortalStatusClosed
	p.TerminationReason = TerminationManualClose
	closedAt := now
	p.ClosedAt = &closedAt
	p.UpdatedAt = now
	return nil
}
