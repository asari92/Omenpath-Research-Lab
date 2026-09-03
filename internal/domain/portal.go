package domain

import (
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
func (p Portal) ScheduledRemaining(now time.Time) time.Duration {
	remaining := p.ScheduledCloseAt.Sub(now)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CurrentEnergy returns the derived realtime energy
// max(0, energy_base − elapsed × decay) (Final Spec §9).
// No per-second storage happens anywhere: callers compute on demand.
func (p Portal) CurrentEnergy(now time.Time) float64 {
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
// ENERGY_DEPLETED — an arbitrary but fixed choice that cannot arise from
// the valid factory (which schedules the hidden moment strictly inside
// the TTL window).
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
