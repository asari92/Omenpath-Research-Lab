package domain

import (
	"fmt"
	"time"

	"omenpath-lab/internal/config"
)

type TutorialSignal string

const (
	TutorialSignalIntroCompleted      TutorialSignal = "TUTORIAL_INTRO_COMPLETED"
	TutorialSignalPortalDetailsOpened TutorialSignal = "PORTAL_DETAILS_OPENED"
	TutorialSignalEventLogOpened      TutorialSignal = "EVENT_LOG_OPENED"
)

func IsTutorialSignal(signal TutorialSignal) bool {
	switch signal {
	case TutorialSignalIntroCompleted, TutorialSignalPortalDetailsOpened, TutorialSignalEventLogOpened:
		return true
	default:
		return false
	}
}

type TutorialPhase string

const (
	TutorialPhaseNone            TutorialPhase = ""
	TutorialPhaseSendReplacement TutorialPhase = "SEND_REPLACEMENT"
	TutorialPhaseWaitResearch    TutorialPhase = "WAIT_RESEARCH"
	TutorialPhaseRecallReady     TutorialPhase = "RECALL_READY"
)

type TutorialExpectedAction string

const (
	TutorialActionCompleteIntro TutorialExpectedAction = "COMPLETE_INTRO"
	TutorialActionOpenDetails   TutorialExpectedAction = "OPEN_PORTAL_DETAILS"
	TutorialActionWaitCorridor  TutorialExpectedAction = "WAIT_CORRIDOR"
	TutorialActionSend          TutorialExpectedAction = "SEND_OBSERVER"
	TutorialActionStabilize     TutorialExpectedAction = "STABILIZE"
	TutorialActionCriticalSend  TutorialExpectedAction = "ATTEMPT_CRITICAL_SEND"
	TutorialActionWaitResearch  TutorialExpectedAction = "WAIT_RESEARCH"
	TutorialActionRecall        TutorialExpectedAction = "RECALL_OBSERVER"
	TutorialActionWaitReturn    TutorialExpectedAction = "WAIT_RETURN"
	TutorialActionOpenEventLog  TutorialExpectedAction = "OPEN_EVENT_LOG"
	TutorialActionStartLive     TutorialExpectedAction = "START_LIVE"
)

func ExpectedTutorialAction(app AppState) (TutorialExpectedAction, error) {
	if app.Mode != ModeTutorial || app.TutorialStep < 0 || app.TutorialStep > 9 {
		return "", ErrSimulationInvariant
	}
	switch app.TutorialStep {
	case 0:
		return TutorialActionCompleteIntro, nil
	case 1:
		return TutorialActionOpenDetails, nil
	case 2:
		return TutorialActionWaitCorridor, nil
	case 3:
		return TutorialActionSend, nil
	case 4:
		return TutorialActionStabilize, nil
	case 5:
		return TutorialActionCriticalSend, nil
	case 6:
		switch app.TutorialPhase {
		case TutorialPhaseSendReplacement:
			return TutorialActionSend, nil
		case TutorialPhaseWaitResearch:
			return TutorialActionWaitResearch, nil
		case TutorialPhaseRecallReady:
			return TutorialActionRecall, nil
		default:
			return "", ErrSimulationInvariant
		}
	case 7:
		return TutorialActionWaitReturn, nil
	case 8:
		return TutorialActionOpenEventLog, nil
	case 9:
		return TutorialActionStartLive, nil
	default:
		return "", ErrSimulationInvariant
	}
}

type TutorialPortalProfile string

const (
	TutorialPortalStep1         TutorialPortalProfile = "STEP_1"
	TutorialPortalStep2         TutorialPortalProfile = "STEP_2"
	TutorialPortalStep3         TutorialPortalProfile = "STEP_3"
	TutorialPortalStep4         TutorialPortalProfile = "STEP_4"
	TutorialPortalStep4High     TutorialPortalProfile = "STEP_4_HIGH"
	TutorialPortalStep4Critical TutorialPortalProfile = "STEP_4_CRITICAL"
	TutorialPortalStep5         TutorialPortalProfile = "STEP_5"
	TutorialPortalStep6Outbound TutorialPortalProfile = "STEP_6_OUTBOUND"
	TutorialPortalStep6Return   TutorialPortalProfile = "STEP_6_RETURN"
)

// NewTutorialPortal creates a deterministic scenario portal from semantic
// properties. It deliberately consumes no random values.
func NewTutorialPortal(profile TutorialPortalProfile, id, planeID int64, slot int, now time.Time, cfg config.Config) (Portal, error) {
	if id <= 0 || planeID <= 0 || slot < 1 || slot > cfg.MaxActivePortals || now.IsZero() || now.Location() != time.UTC {
		return Portal{}, ErrSimulationInvariant
	}
	portal := Portal{
		ID: id, Name: fmt.Sprintf("Omenpath #%04d", id), SlotIndex: slot,
		Kind: PortalKindNatural, DestinationPlaneID: planeID,
		EnergyBase: 100, EnergyBaseAt: now, EnergyDecayRate: 0.1,
		Stability: PortalStable, OpenedAt: now, ScheduledCloseAt: now.Add(2 * time.Minute),
		ObserverFlow: PortalFlowNone, Status: PortalStatusOpen,
		CreatedAt: now, UpdatedAt: now,
	}
	switch profile {
	case TutorialPortalStep1, TutorialPortalStep2:
		portal.CreaturesInitial = 3
	case TutorialPortalStep4, TutorialPortalStep4High:
		portal.EnergyBase = 25
		portal.EnergyDecayRate = 1
		portal.Stability = PortalUnstable
		hidden := now.Add(119 * time.Second)
		portal.InstabilityCollapseAt = &hidden
	case TutorialPortalStep4Critical:
		portal.EnergyBase = 19
		portal.EnergyDecayRate = 1
		portal.Stability = PortalUnstable
		hidden := now.Add(119 * time.Second)
		portal.InstabilityCollapseAt = &hidden
	case TutorialPortalStep5:
		portal.ScheduledCloseAt = now.Add(10 * time.Second)
	case TutorialPortalStep3, TutorialPortalStep6Outbound, TutorialPortalStep6Return:
		portal.ScheduledCloseAt = now.Add(2 * time.Minute)
	default:
		return Portal{}, ErrSimulationInvariant
	}
	return portal, nil
}
