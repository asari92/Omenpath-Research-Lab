package domain

import (
	"fmt"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/random"
)

// NewNaturalPortal creates a valid NATURAL portal (Final Spec §7, §9, §10,
// §12). It MUST NOT choose the plane or the slot, mutate lab state, write
// the database, emit events or broadcast — that is later orchestration
// (Stage 2 plan §20). The caller passes the already-chosen slot and the
// sequential number used for the ID and the `Omenpath #XXXX` name
// (PORTAL-002).
//
// Deterministic draw order (for FakeRandom fixtures):
//
//  1. TTL seconds            IntInclusive(NaturalTTLMin, NaturalTTLMax)
//  2. initial energy         FloatRange(PortalEnergyMin, PortalEnergyMax)
//  3. decay rate             FloatRange(PortalDecayMin, PortalDecayMax)
//  4. stability roll         FloatRange(0, 1) < UnstableProbability
//  5. hidden collapse offset IntInclusive(0, window) — only when unstable
//  6. creatures              IntInclusive(0, MaxCreaturesForTTL(ttl))
func NewNaturalPortal(seq, planeID int64, slot int, now time.Time, cfg config.Config, rnd random.Random) Portal {
	ttl := time.Duration(rnd.IntInclusive(
		int(cfg.NaturalTTLMin.Seconds()),
		int(cfg.NaturalTTLMax.Seconds()),
	)) * time.Second
	energy := rnd.FloatRange(cfg.PortalEnergyMin, cfg.PortalEnergyMax)
	decay := rnd.FloatRange(cfg.PortalDecayMin, cfg.PortalDecayMax)
	unstable := rnd.FloatRange(0, 1) < cfg.UnstableProbability

	scheduledClose := now.Add(ttl)

	var hidden *time.Time
	if unstable {
		// Final Spec §10 / resolved decision S2-D1: the hidden collapse is
		// scheduled inside [opened_at + InstabilityMinLifetime,
		// scheduled_close_at − InstabilityCloseMargin]. Valid config keeps
		// this window non-empty (min TTL 10s > 5s + 1s).
		lo := now.Add(cfg.InstabilityMinLifetime)
		hi := scheduledClose.Add(-cfg.InstabilityCloseMargin)
		offset := time.Duration(rnd.IntInclusive(0, int(hi.Sub(lo))))
		at := lo.Add(offset)
		hidden = &at
	}

	creatures := rnd.IntInclusive(0, MaxCreaturesForTTL(ttl, cfg))

	stability := PortalStable
	if unstable {
		stability = PortalUnstable
	}

	return Portal{
		ID:                 seq,
		Name:               fmt.Sprintf("Omenpath #%04d", seq),
		SlotIndex:          slot,
		Kind:               PortalKindNatural,
		DestinationPlaneID: planeID,

		EnergyBase:      energy,
		EnergyBaseAt:    now,
		EnergyDecayRate: decay,

		Stability:             stability,
		InstabilityCollapseAt: hidden,

		OpenedAt:         now,
		ScheduledCloseAt: scheduledClose,

		CreaturesInitial: creatures,

		ObserverFlow: PortalFlowNone,

		Status:            PortalStatusOpen,
		TerminationReason: TerminationNone,

		CreatedAt: now,
		UpdatedAt: now,
	}
}
