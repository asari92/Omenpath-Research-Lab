package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
)

// StartTutorial is intentionally idempotent: a bootstrapped or partially
// completed Tutorial is continued exactly where persistence left it.
func (m *LabManager) StartTutorial(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("start tutorial: nil manager")
	}
	if err := m.mu.LockContext(ctx); err != nil {
		return err
	}
	defer m.mu.Unlock()
	if m.snapshot.App.Mode == domain.ModeTutorial {
		return nil
	}
	return m.commitTutorialRejectionLocked(ctx, "START_TUTORIAL", ErrTutorialNotReady)
}

func (m *LabManager) ResetTutorial(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("reset tutorial: nil manager")
	}
	if err := m.mu.LockContext(ctx); err != nil {
		return err
	}
	defer m.mu.Unlock()
	now := m.clock.Now().UTC()
	reset := canonicalTutorialSnapshot(m.snapshot, now, m.cfg.ObserverCount, m.cfg.LabEnergyMax)
	if err := m.repo.ResetTutorial(ctx, cloneSnapshot(reset)); err != nil {
		return fmt.Errorf("reset tutorial: %w", err)
	}
	m.snapshot = cloneSnapshot(reset)
	m.signalLocked()
	return nil
}

func canonicalTutorialSnapshot(current persistence.Snapshot, now time.Time, observerCount, labMaximum int) persistence.Snapshot {
	planes := append([]domain.Plane(nil), current.Simulation.Planes...)
	for i := range planes {
		planes[i].Aliases = append([]string(nil), planes[i].Aliases...)
		planes[i].Explored = false
		planes[i].ExploredAt = nil
	}
	return persistence.Snapshot{
		Simulation: domain.SimulationState{
			Lab:          domain.LabState{EnergyBase: labMaximum, EnergyBaseAt: now},
			Planes:       planes,
			Observers:    domain.NewObserverRoster(observerCount, now),
			NextPortalID: 1,
			NaturalSpawn: domain.NaturalSpawnState{Paused: true},
		},
		App: domain.AppState{Mode: domain.ModeTutorial},
	}
}

func (m *LabManager) StartLive(ctx context.Context) (err error) {
	if m == nil {
		return fmt.Errorf("start live: nil manager")
	}
	if err := m.mu.LockContext(ctx); err != nil {
		return err
	}
	defer m.mu.Unlock()
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
	resolvedBeforeTutorial := cloneSnapshot(working)
	if err := m.advanceTutorialAfterTick(&working, now); err != nil {
		return err
	}
	lifecycleDrafts := cloneDrafts(tick.Events)
	if !reflect.DeepEqual(resolvedBeforeTutorial.Simulation, working.Simulation) {
		tutorialDrafts, eventErr := domain.EventsForStateTransition(
			resolvedBeforeTutorial.Simulation, working.Simulation, now, now, m.cfg,
		)
		if eventErr != nil {
			return eventErr
		}
		lifecycleDrafts = append(lifecycleDrafts, tutorialDrafts...)
	}
	if working.App.Mode != domain.ModeTutorial || working.App.TutorialStep != 9 {
		rejected, eventErr := domain.NewActionRejectedEvent(now, "START_LIVE", nil, nil, nil, ErrTutorialNotReady)
		if eventErr != nil {
			return eventErr
		}
		drafts := append(cloneDrafts(lifecycleDrafts), rejected)
		if _, err := m.repo.Commit(ctx, cloneSnapshot(working), drafts); err != nil {
			return fmt.Errorf("commit START_LIVE rejection: %w", err)
		}
		m.snapshot = cloneSnapshot(working)
		randomTx.commit()
		m.signalLocked()
		return ErrTutorialNotReady
	}

	beforeClose := cloneSnapshot(working)
	for i := range working.Simulation.Portals {
		if working.Simulation.Portals[i].Status != domain.PortalStatusOpen {
			continue
		}
		if err := working.Simulation.Portals[i].Close(now, true, m.cfg); err != nil {
			return err
		}
	}
	for i := range working.Simulation.Observers {
		observer := &working.Simulation.Observers[i]
		if observer.Status != domain.ObserverOutbound && observer.Status != domain.ObserverReturning {
			continue
		}
		if observer.ActivePortalID == nil {
			return domain.ErrSimulationInvariant
		}
		portalAt, ok := portalIndex(working.Simulation.Portals, *observer.ActivePortalID)
		if !ok {
			return domain.ErrSimulationInvariant
		}
		planeAt, ok := planeIndex(working.Simulation.Planes, working.Simulation.Portals[portalAt].DestinationPlaneID)
		if !ok {
			return domain.ErrSimulationInvariant
		}
		if err := domain.ResolveObserverLifecycle(
			observer,
			&working.Simulation.Planes[planeAt],
			&working.Simulation.Portals[portalAt],
			now,
			m.cfg,
		); err != nil {
			return err
		}
	}
	closeDrafts, err := domain.EventsForStateTransition(beforeClose.Simulation, working.Simulation, now, now, m.cfg)
	if err != nil {
		return err
	}
	for i := range closeDrafts {
		if closeDrafts[i].EventType != domain.EventPortalClosed {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(closeDrafts[i].PayloadJSON), &payload); err != nil {
			return domain.ErrEventInvariant
		}
		payload["source"] = "TUTORIAL_COMPLETION"
		encoded, err := json.Marshal(payload)
		if err != nil {
			return domain.ErrEventInvariant
		}
		closeDrafts[i].PayloadJSON = string(encoded)
	}
	spawn, err := domain.NewNaturalSpawnState(now, m.cfg, m.random)
	if err != nil {
		return err
	}
	working.Simulation.NaturalSpawn = spawn
	working.App = domain.AppState{Mode: domain.ModeLive, TutorialStep: 9}
	drafts := append(cloneDrafts(lifecycleDrafts), closeDrafts...)
	if _, err := m.repo.Commit(ctx, cloneSnapshot(working), drafts); err != nil {
		return fmt.Errorf("commit START_LIVE: %w", err)
	}
	m.snapshot = cloneSnapshot(working)
	randomTx.commit()
	m.signalLocked()
	return nil
}

func (m *LabManager) commitTutorialRejectionLocked(ctx context.Context, action string, cause error) (err error) {
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
	rejected, err := domain.NewActionRejectedEvent(now, action, nil, nil, nil, cause)
	if err != nil {
		return err
	}
	drafts := append(cloneDrafts(tick.Events), rejected)
	if _, err := m.repo.Commit(ctx, cloneSnapshot(working), drafts); err != nil {
		return fmt.Errorf("commit %s rejection: %w", action, err)
	}
	m.snapshot = cloneSnapshot(working)
	randomTx.commit()
	m.signalLocked()
	return cause
}
