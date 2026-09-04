package engine

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
)

func (m *LabManager) TutorialSignal(ctx context.Context, signal domain.TutorialSignal, portalID *int64) error {
	if m == nil {
		return fmt.Errorf("tutorial signal: nil manager")
	}
	if err := m.mu.LockContext(ctx); err != nil {
		return err
	}
	defer m.mu.Unlock()
	return m.executeTutorialSignalLocked(ctx, signal, portalID)
}

func (m *LabManager) executeTutorialSignalLocked(ctx context.Context, signal domain.TutorialSignal, portalID *int64) (err error) {
	randomTx, err := beginRandomTransaction(m.random)
	if err != nil {
		return fmt.Errorf("checkpoint random state: %w", err)
	}
	defer func() { err = randomTx.finish(err) }()
	now := m.clock.Now().UTC()
	working := cloneSnapshot(m.snapshot)
	prepareTutorialSpawn(&working)
	tick, err := working.Simulation.ResolveTick(now, m.random, m.cfg)
	if err != nil {
		return err
	}
	resolved := cloneSnapshot(working)
	if err := m.advanceTutorialAfterTick(&working, now); err != nil {
		return err
	}
	if !reflect.DeepEqual(resolved.Simulation, working.Simulation) {
		tutorialDrafts, eventErr := domain.EventsForStateTransition(resolved.Simulation, working.Simulation, now, now, m.cfg)
		if eventErr != nil {
			return eventErr
		}
		tick.Events = append(tick.Events, tutorialDrafts...)
	}
	resolved = cloneSnapshot(working)
	signalErr := applyTutorialSignal(&working, signal, portalID, now, m.cfg)
	drafts := cloneDrafts(tick.Events)
	if signalErr == nil {
		created, eventErr := domain.EventsForStateTransition(resolved.Simulation, working.Simulation, now, now, m.cfg)
		if eventErr != nil {
			return eventErr
		}
		drafts = append(drafts, created...)
	} else {
		rejected, eventErr := domain.NewActionRejectedEvent(now, "TUTORIAL_SIGNAL", portalID, nil, nil, signalErr)
		if eventErr != nil {
			return eventErr
		}
		drafts = append(drafts, rejected)
	}
	if _, err := m.repo.Commit(ctx, cloneSnapshot(working), cloneDrafts(drafts)); err != nil {
		return fmt.Errorf("commit tutorial signal: %w", err)
	}
	m.snapshot = cloneSnapshot(working)
	randomTx.commit()
	m.signalLocked()
	return signalErr
}

func applyTutorialSignal(snapshot *persistence.Snapshot, signal domain.TutorialSignal, portalID *int64, now time.Time, cfg config.Config) error {
	if snapshot.App.Mode != domain.ModeTutorial || !domain.IsTutorialSignal(signal) {
		return ErrInvalidTutorialSignal
	}
	switch {
	case snapshot.App.TutorialStep == 0 && signal == domain.TutorialSignalIntroCompleted && portalID == nil:
		if err := createTutorialTarget(snapshot, domain.TutorialPortalStep1, 0, now, cfg); err != nil {
			return err
		}
		snapshot.App.TutorialStep = 1
		return nil
	case snapshot.App.TutorialStep == 1 && signal == domain.TutorialSignalPortalDetailsOpened &&
		portalID != nil && snapshot.App.TutorialPortalID != nil && *portalID == *snapshot.App.TutorialPortalID:
		snapshot.App.TutorialStep = 2
		return nil
	case snapshot.App.TutorialStep == 8 && signal == domain.TutorialSignalEventLogOpened && portalID == nil:
		snapshot.App.TutorialStep = 9
		return nil
	default:
		return ErrInvalidTutorialSignal
	}
}

func prepareTutorialSpawn(snapshot *persistence.Snapshot) {
	if snapshot.App.Mode == domain.ModeTutorial {
		snapshot.Simulation.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	}
}

func (m *LabManager) advanceTutorialAfterTick(snapshot *persistence.Snapshot, now time.Time) error {
	if snapshot.App.Mode != domain.ModeTutorial {
		return nil
	}
	prepareTutorialSpawn(snapshot)
	if snapshot.App.TutorialStep == 0 {
		return nil
	}
	if err := recreateTerminalTutorialTarget(snapshot, now, m.cfg); err != nil {
		return err
	}
	if snapshot.App.TutorialStep == 2 {
		portal, ok := tutorialTarget(snapshot)
		if !ok {
			return domain.ErrSimulationInvariant
		}
		if portal.Status == domain.PortalStatusOpen && portal.CreaturesInside(now, m.cfg) == 0 {
			snapshot.App.TutorialStep = 3
		}
	}
	if snapshot.App.TutorialStep == 6 {
		observer := trackedObserver(snapshot)
		if observer != nil && observer.Status == domain.ObserverLost {
			if snapshot.App.TutorialPhase == domain.TutorialPhaseSendReplacement {
				if target, ok := tutorialTarget(snapshot); ok && target.Status == domain.PortalStatusOpen {
					return nil
				}
			}
			snapshot.App.TutorialPhase = domain.TutorialPhaseSendReplacement
			return createTutorialTarget(snapshot, domain.TutorialPortalStep6Outbound, pointerValue(snapshot.App.TutorialPlaneID), now, m.cfg)
		}
		if snapshot.App.TutorialPhase == domain.TutorialPhaseWaitResearch && observer != nil && observer.Status == domain.ObserverWaitingReturn {
			if observer.CurrentPlaneID == nil {
				return domain.ErrSimulationInvariant
			}
			if err := createTutorialTarget(snapshot, domain.TutorialPortalStep6Return, *observer.CurrentPlaneID, now, m.cfg); err != nil {
				return err
			}
			snapshot.App.TutorialPhase = domain.TutorialPhaseRecallReady
		}
	}
	if snapshot.App.TutorialStep == 7 {
		observer := trackedObserver(snapshot)
		if observer == nil {
			return domain.ErrSimulationInvariant
		}
		switch observer.Status {
		case domain.ObserverLost:
			snapshot.App.TutorialStep = 6
			snapshot.App.TutorialPhase = domain.TutorialPhaseSendReplacement
			return createTutorialTarget(snapshot, domain.TutorialPortalStep6Outbound, pointerValue(snapshot.App.TutorialPlaneID), now, m.cfg)
		case domain.ObserverAvailable:
			planeAt, ok := planeIndex(snapshot.Simulation.Planes, pointerValue(snapshot.App.TutorialPlaneID))
			if ok && snapshot.Simulation.Planes[planeAt].Explored {
				snapshot.App.TutorialStep = 8
				snapshot.App.TutorialPhase = domain.TutorialPhaseNone
				snapshot.App.TutorialPortalID = nil
			}
		}
	}
	return nil
}

func (m *LabManager) advanceTutorialAfterCommand(snapshot, before *persistence.Snapshot, command managerCommand, commandErr error, now time.Time) error {
	if snapshot.App.Mode != domain.ModeTutorial {
		return nil
	}
	prepareTutorialSpawn(snapshot)
	matched := command.portalID != nil && snapshot.App.TutorialPortalID != nil && *command.portalID == *snapshot.App.TutorialPortalID
	switch snapshot.App.TutorialStep {
	case 3:
		if commandErr == nil && command.action == "SEND" && matched {
			observer := dispatchedThrough(snapshot.Simulation.Observers, *command.portalID)
			if observer == nil {
				return domain.ErrSimulationInvariant
			}
			portal, ok := tutorialTarget(snapshot)
			if !ok {
				return domain.ErrSimulationInvariant
			}
			snapshot.App.TutorialObserverID = int64Pointer(observer.ID)
			snapshot.App.TutorialPlaneID = int64Pointer(portal.DestinationPlaneID)
			if err := createTutorialTarget(snapshot, domain.TutorialPortalStep4, portal.DestinationPlaneID, now, m.cfg); err != nil {
				return err
			}
			snapshot.App.TutorialStep = 4
		}
	case 4:
		if commandErr == nil && command.action == "STABILIZE" && matched {
			oldPortal, oldOK := tutorialTarget(before)
			newPortal, newOK := tutorialTarget(snapshot)
			if !oldOK || !newOK || oldPortal.Stability != domain.PortalUnstable || newPortal.Stability != domain.PortalStable {
				return domain.ErrSimulationInvariant
			}
			beforeRisk, beforeOK := oldPortal.RiskLevel(now, m.cfg)
			afterRisk, afterOK := newPortal.RiskLevel(now, m.cfg)
			if !beforeOK || !afterOK || !containsRisk(beforeRisk, domain.RiskHigh, domain.RiskCritical) || !containsRisk(afterRisk, domain.RiskLow, domain.RiskMedium) {
				return domain.ErrSimulationInvariant
			}
			if err := createTutorialTarget(snapshot, domain.TutorialPortalStep5, 0, now, m.cfg); err != nil {
				return err
			}
			snapshot.App.TutorialStep = 5
		}
	case 5:
		if command.action == "SEND" && matched && errors.Is(commandErr, domain.ErrPortalCriticalRisk) {
			snapshot.App.TutorialStep = 6
			return enterTutorialStep6(snapshot, now, m.cfg)
		}
	case 6:
		if commandErr == nil && command.action == "SEND" && matched && snapshot.App.TutorialPhase == domain.TutorialPhaseSendReplacement {
			observer := dispatchedThrough(snapshot.Simulation.Observers, *command.portalID)
			if observer == nil {
				return domain.ErrSimulationInvariant
			}
			snapshot.App.TutorialObserverID = int64Pointer(observer.ID)
			snapshot.App.TutorialPhase = domain.TutorialPhaseWaitResearch
		}
		if commandErr == nil && command.action == "RECALL" && matched && snapshot.App.TutorialPhase == domain.TutorialPhaseRecallReady {
			observer := returningThrough(snapshot.Simulation.Observers, *command.portalID)
			if observer == nil {
				return domain.ErrSimulationInvariant
			}
			snapshot.App.TutorialObserverID = int64Pointer(observer.ID)
			snapshot.App.TutorialStep = 7
			snapshot.App.TutorialPhase = domain.TutorialPhaseNone
		}
	}
	return recreateTerminalTutorialTarget(snapshot, now, m.cfg)
}

func containsRisk(value domain.RiskLevel, allowed ...domain.RiskLevel) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func enterTutorialStep6(snapshot *persistence.Snapshot, now time.Time, cfg config.Config) error {
	observer := trackedObserver(snapshot)
	if observer != nil {
		switch observer.Status {
		case domain.ObserverOutbound, domain.ObserverExploring:
			snapshot.App.TutorialPhase = domain.TutorialPhaseWaitResearch
			return nil
		case domain.ObserverWaitingReturn:
			snapshot.App.TutorialPhase = domain.TutorialPhaseRecallReady
			return createTutorialTarget(snapshot, domain.TutorialPortalStep6Return, *observer.CurrentPlaneID, now, cfg)
		}
	}
	snapshot.App.TutorialPhase = domain.TutorialPhaseSendReplacement
	planeID := pointerValue(snapshot.App.TutorialPlaneID)
	return createTutorialTarget(snapshot, domain.TutorialPortalStep6Outbound, planeID, now, cfg)
}

func recreateTerminalTutorialTarget(snapshot *persistence.Snapshot, now time.Time, cfg config.Config) error {
	portal, ok := tutorialTarget(snapshot)
	if !ok || portal.Status == domain.PortalStatusOpen {
		return nil
	}
	var profile domain.TutorialPortalProfile
	switch snapshot.App.TutorialStep {
	case 1:
		profile = domain.TutorialPortalStep1
	case 2:
		profile = domain.TutorialPortalStep2
	case 3:
		profile = domain.TutorialPortalStep3
	case 4:
		profile = domain.TutorialPortalStep4
	case 5:
		profile = domain.TutorialPortalStep5
	case 6:
		if snapshot.App.TutorialPhase == domain.TutorialPhaseRecallReady {
			profile = domain.TutorialPortalStep6Return
		} else if snapshot.App.TutorialPhase == domain.TutorialPhaseSendReplacement {
			profile = domain.TutorialPortalStep6Outbound
		} else {
			return nil
		}
	default:
		return nil
	}
	return createTutorialTarget(snapshot, profile, portal.DestinationPlaneID, now, cfg)
}

func createTutorialTarget(snapshot *persistence.Snapshot, profile domain.TutorialPortalProfile, planeID int64, now time.Time, cfg config.Config) error {
	if planeID == 0 {
		planeID = firstTutorialPlane(snapshot.Simulation.Planes)
	}
	if planeID == 0 {
		return domain.ErrSimulationInvariant
	}
	slot, ok := domain.FirstFreeSlot(snapshot.Simulation.Portals, cfg.MaxActivePortals)
	if !ok {
		return domain.ErrNoFreePortalSlot
	}
	portal, err := domain.NewTutorialPortal(profile, snapshot.Simulation.NextPortalID, planeID, slot, now, cfg)
	if err != nil {
		return err
	}
	snapshot.Simulation.Portals = append(snapshot.Simulation.Portals, portal)
	snapshot.Simulation.NextPortalID++
	snapshot.App.TutorialPortalID = int64Pointer(portal.ID)
	snapshot.App.TutorialPlaneID = int64Pointer(planeID)
	return nil
}

func firstTutorialPlane(planes []domain.Plane) int64 {
	ids := make([]int64, 0, len(planes))
	for _, plane := range planes {
		if !plane.Explored {
			ids = append(ids, plane.ID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func tutorialTarget(snapshot *persistence.Snapshot) (*domain.Portal, bool) {
	if snapshot.App.TutorialPortalID == nil {
		return nil, false
	}
	index, ok := portalIndex(snapshot.Simulation.Portals, *snapshot.App.TutorialPortalID)
	if !ok {
		return nil, false
	}
	return &snapshot.Simulation.Portals[index], true
}

func trackedObserver(snapshot *persistence.Snapshot) *domain.Observer {
	if snapshot.App.TutorialObserverID == nil {
		return nil
	}
	for i := range snapshot.Simulation.Observers {
		if snapshot.Simulation.Observers[i].ID == *snapshot.App.TutorialObserverID {
			return &snapshot.Simulation.Observers[i]
		}
	}
	return nil
}

func dispatchedThrough(observers []domain.Observer, portalID int64) *domain.Observer {
	for i := range observers {
		if observers[i].Status == domain.ObserverOutbound && observers[i].ActivePortalID != nil && *observers[i].ActivePortalID == portalID {
			return &observers[i]
		}
	}
	return nil
}

func returningThrough(observers []domain.Observer, portalID int64) *domain.Observer {
	for i := range observers {
		if observers[i].Status == domain.ObserverReturning && observers[i].ActivePortalID != nil && *observers[i].ActivePortalID == portalID {
			return &observers[i]
		}
	}
	return nil
}

func int64Pointer(value int64) *int64 { return &value }

func pointerValue(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
