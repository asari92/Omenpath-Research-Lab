package testutil

import (
	"time"

	"omenpath-lab/internal/domain"
)

// PortalBuilder builds Portal fixtures for domain tests.
// It is a test helper, NOT a production constructor: factories with
// validation arrive in Stage 2. Defaults describe a boring, safe portal.
type PortalBuilder struct {
	portal domain.Portal
}

// NewPortalBuilder returns a builder with deterministic defaults:
// NATURAL OPEN STABLE portal opened at BaseTime, TTL 60s,
// energy 100 with decay 0.5/sec, no creatures, flow NONE.
func NewPortalBuilder() *PortalBuilder {
	opened := BaseTime
	return &PortalBuilder{portal: domain.Portal{
		ID:                 1,
		Name:               "Omenpath #0001",
		SlotIndex:          1,
		Kind:               domain.PortalKindNatural,
		DestinationPlaneID: 1,

		EnergyBase:      100,
		EnergyBaseAt:    opened,
		EnergyDecayRate: 0.5,

		Stability:             domain.PortalStable,
		InstabilityCollapseAt: nil,

		OpenedAt:         opened,
		ScheduledCloseAt: opened.Add(60 * time.Second),

		CreaturesInitial: 0,

		ObserverFlow: domain.PortalFlowNone,

		Status:            domain.PortalStatusOpen,
		TerminationReason: domain.TerminationNone,

		CreatedAt: opened,
		UpdatedAt: opened,
	}}
}

// Stable marks the portal STABLE with no hidden collapse timestamp.
func (b *PortalBuilder) Stable() *PortalBuilder {
	b.portal.Stability = domain.PortalStable
	b.portal.InstabilityCollapseAt = nil
	return b
}

// Unstable marks the portal UNSTABLE with an explicit hidden collapse moment.
// The caller decides the timestamp — hidden-collapse semantics are Stage 2
// business, so fixtures stay explicit.
func (b *PortalBuilder) Unstable(hiddenCollapseAt time.Time) *PortalBuilder {
	b.portal.Stability = domain.PortalUnstable
	at := hiddenCollapseAt
	b.portal.InstabilityCollapseAt = &at
	return b
}

// Energy sets the energy baseline (%).
func (b *PortalBuilder) Energy(v float64) *PortalBuilder {
	b.portal.EnergyBase = v
	return b
}

// Decay sets the energy decay rate (%/sec).
func (b *PortalBuilder) Decay(v float64) *PortalBuilder {
	b.portal.EnergyDecayRate = v
	return b
}

// TTL sets ScheduledCloseAt = OpenedAt + d.
func (b *PortalBuilder) TTL(d time.Duration) *PortalBuilder {
	b.portal.ScheduledCloseAt = b.portal.OpenedAt.Add(d)
	return b
}

// Creatures sets the initial creature count.
func (b *PortalBuilder) Creatures(n int) *PortalBuilder {
	b.portal.CreaturesInitial = n
	return b
}

// Slot sets the fixed slot index (1-based, Final Spec §5).
func (b *PortalBuilder) Slot(n int) *PortalBuilder {
	b.portal.SlotIndex = n
	return b
}

// Build returns a copy of the fixture.
func (b *PortalBuilder) Build() domain.Portal {
	return b.portal
}
