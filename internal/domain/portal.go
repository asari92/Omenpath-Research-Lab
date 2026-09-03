package domain

import "time"

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
