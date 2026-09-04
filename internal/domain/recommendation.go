package domain

import (
	"time"

	"omenpath-lab/internal/config"
)

type Recommendation string

const (
	RecommendationLeaveOpen       Recommendation = "LEAVE OPEN"
	RecommendationWaitForCorridor Recommendation = "WAIT FOR CORRIDOR"
	RecommendationStabilize       Recommendation = "STABILIZE"
	RecommendationClose           Recommendation = "CLOSE"
	RecommendationSendObserver    Recommendation = "SEND OBSERVER"
	RecommendationRecallObserver  Recommendation = "RECALL OBSERVER"
)

// RecommendationForPortal applies the ordered, deterministic safety table
// from Final Spec §23.1. It only inspects a value copy and never participates
// in command eligibility or mutation.
func RecommendationForPortal(state SimulationState, portalID int64, now time.Time, cfg config.Config) (Recommendation, bool, error) {
	if err := ValidateSimulationStructure(state, now); err != nil || !validRecommendationConfig(cfg) {
		return "", false, ErrSimulationInvariant
	}
	portalIndex := -1
	for i := range state.Portals {
		if state.Portals[i].ID == portalID {
			portalIndex = i
			break
		}
	}
	if portalIndex == -1 {
		return "", false, ErrSimulationInvariant
	}
	portal := state.Portals[portalIndex]
	if portal.Status != PortalStatusOpen {
		return "", false, nil
	}
	if portal.Kind == PortalKindExtraction && portal.ExtractionSynchronizedAt == nil {
		return RecommendationLeaveOpen, true, nil
	}

	if activeIndex, active, err := ActiveTransitObserverIndex(state.Observers, portal.ID, now); err != nil {
		return "", false, err
	} else if active {
		horizon := state.Observers[activeIndex].PhaseEndsAt.Sub(now)
		if portalSafeFor(portal, horizon, now, cfg) {
			return RecommendationLeaveOpen, true, nil
		}
		if stabilizationMakesSafe(state.Lab, portal, horizon, now, cfg) {
			return RecommendationStabilize, true, nil
		}
		return RecommendationLeaveOpen, true, nil
	}

	flowAllowsInbound := portal.ObserverFlow == PortalFlowNone || portal.ObserverFlow == PortalFlowInbound
	if flowAllowsInbound {
		if _, waiting, err := LongestWaitingObserverIndex(state.Observers, portal.DestinationPlaneID); err != nil {
			return "", false, err
		} else if waiting {
			return recommendationForImmediateMission(state, portal, now, cfg, RecommendationRecallObserver)
		}
		if remaining, exploring, err := shortestResearchRemaining(state.Observers, portal.DestinationPlaneID, now); err != nil {
			return "", false, err
		} else if exploring {
			horizon := remaining + cfg.ObserverTransitMax
			if portalSafeFor(portal, horizon, now, cfg) {
				return RecommendationLeaveOpen, true, nil
			}
			if stabilizationMakesSafe(state.Lab, portal, horizon, now, cfg) {
				return RecommendationStabilize, true, nil
			}
			return recommendationFallback(state, portal, now, cfg), true, nil
		}
	}

	plane, ok := recommendationPlane(state.Planes, portal.DestinationPlaneID)
	if !ok {
		return "", false, ErrSimulationInvariant
	}
	_, available := AvailableObserverIndex(state.Observers)
	canUseOutbound := portal.ObserverFlow == PortalFlowNone || portal.ObserverFlow == PortalFlowOutbound
	if !plane.Explored && canUseOutbound && available &&
		!planeHasObserver(state.Observers, plane.ID) &&
		!outboundObserverTargetsPlane(state, plane.ID) {
		return recommendationForImmediateMission(state, portal, now, cfg, RecommendationSendObserver)
	}
	return recommendationFallback(state, portal, now, cfg), true, nil
}

func recommendationForImmediateMission(state SimulationState, portal Portal, now time.Time, cfg config.Config, action Recommendation) (Recommendation, bool, error) {
	clearance := creatureClearanceRemaining(portal, now, cfg)
	if clearance == 0 && portalSafeFor(portal, cfg.ObserverTransitMax, now, cfg) {
		return action, true, nil
	}
	horizon := clearance + cfg.ObserverTransitMax
	if clearance > 0 && portalSafeFor(portal, horizon, now, cfg) {
		return RecommendationWaitForCorridor, true, nil
	}
	if stabilizationMakesSafe(state.Lab, portal, horizon, now, cfg) {
		return RecommendationStabilize, true, nil
	}
	return recommendationFallback(state, portal, now, cfg), true, nil
}

func recommendationFallback(state SimulationState, portal Portal, now time.Time, cfg config.Config) Recommendation {
	plane, _ := recommendationPlane(state.Planes, portal.DestinationPlaneID)
	if plane.Explored && !planeHasObserver(state.Observers, plane.ID) && closeAffordable(state.Lab, now, cfg) {
		return RecommendationClose
	}
	risk, _ := portal.RiskLevel(now, cfg)
	if (risk == RiskHigh || risk == RiskCritical) && closeAffordable(state.Lab, now, cfg) {
		return RecommendationClose
	}
	return RecommendationLeaveOpen
}

func portalSafeFor(portal Portal, horizon time.Duration, now time.Time, cfg config.Config) bool {
	if portal.Status != PortalStatusOpen || portal.Stability != PortalStable || horizon < 0 {
		return false
	}
	risk, ok := portal.RiskLevel(now, cfg)
	return ok && risk != RiskCritical && portal.EffectiveLifetime(now) > horizon
}

func stabilizationMakesSafe(lab LabState, portal Portal, horizon time.Duration, now time.Time, cfg config.Config) bool {
	if portal.Status != PortalStatusOpen || portal.Stability != PortalUnstable ||
		portal.CurrentEnergy(now) > cfg.StabilizeMaxStartEnergy {
		return false
	}
	labCopy, portalCopy := lab, portal
	if err := StabilizePortalWithLabEnergy(&labCopy, &portalCopy, now, cfg); err != nil {
		return false
	}
	return portalSafeFor(portalCopy, horizon, now, cfg)
}

func closeAffordable(lab LabState, now time.Time, cfg config.Config) bool {
	cost := cfg.CloseCost
	if lab.LeylineOverrideActive(now, cfg) {
		cost = 0
	}
	return lab.CanAfford(now, cost, cfg)
}

func creatureClearanceRemaining(portal Portal, now time.Time, cfg config.Config) time.Duration {
	if portal.CreaturesInside(now, cfg) == 0 {
		return 0
	}
	clearedAt := portal.OpenedAt.Add(time.Duration(portal.CreaturesInitial) * cfg.CreatureTransit)
	if !clearedAt.After(now) {
		return 0
	}
	return clearedAt.Sub(now)
}

func shortestResearchRemaining(observers []Observer, planeID int64, now time.Time) (time.Duration, bool, error) {
	var selected time.Duration
	found := false
	for i := range observers {
		observer := observers[i]
		if observer.Status != ObserverExploring || observer.CurrentPlaneID == nil || *observer.CurrentPlaneID != planeID {
			continue
		}
		if observer.PhaseStartedAt == nil || observer.PhaseEndsAt == nil || observer.PhaseEndsAt.Before(*observer.PhaseStartedAt) {
			return 0, false, ErrObserverInvariant
		}
		remaining := observer.PhaseEndsAt.Sub(now)
		if remaining < 0 {
			remaining = 0
		}
		if !found || remaining < selected {
			selected, found = remaining, true
		}
	}
	return selected, found, nil
}

func recommendationPlane(planes []Plane, id int64) (Plane, bool) {
	for _, plane := range planes {
		if plane.ID == id {
			return plane, true
		}
	}
	return Plane{}, false
}

func planeHasObserver(observers []Observer, planeID int64) bool {
	for _, observer := range observers {
		if observer.CurrentPlaneID != nil && *observer.CurrentPlaneID == planeID && observer.Status != ObserverLost {
			return true
		}
	}
	return false
}

func outboundObserverTargetsPlane(state SimulationState, planeID int64) bool {
	for _, observer := range state.Observers {
		if observer.Status != ObserverOutbound || observer.ActivePortalID == nil {
			continue
		}
		for _, portal := range state.Portals {
			if portal.ID == *observer.ActivePortalID && portal.DestinationPlaneID == planeID {
				return true
			}
		}
	}
	return false
}

func validRecommendationConfig(cfg config.Config) bool {
	return cfg.ObserverTransitMax > 0 && cfg.CreatureTransit > 0 && cfg.RiskSafeHorizon > 0 &&
		cfg.LabEnergyMax > 0 && cfg.CloseCost >= 0 && cfg.StabilizeCost >= 0
}
