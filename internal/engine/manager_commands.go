package engine

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
)

type managerCommand struct {
	action     string
	portalID   *int64
	observerID *int64
	planeID    *int64
	apply      func(*persistence.Snapshot, time.Time) error
}

func (m *LabManager) Stabilize(ctx context.Context, id int64) error {
	return m.executeCommand(ctx, managerCommand{
		action:   "STABILIZE",
		portalID: &id,
		apply: func(snapshot *persistence.Snapshot, now time.Time) error {
			index, ok := portalIndex(snapshot.Simulation.Portals, id)
			if !ok {
				return ErrPortalNotFound
			}
			return domain.StabilizePortalWithLabEnergy(
				&snapshot.Simulation.Lab, &snapshot.Simulation.Portals[index], now, m.cfg,
			)
		},
	})
}

func (m *LabManager) ClosePortal(ctx context.Context, id int64, confirm bool) error {
	return m.executeCommand(ctx, managerCommand{
		action:   "CLOSE",
		portalID: &id,
		apply: func(snapshot *persistence.Snapshot, now time.Time) error {
			portalAt, ok := portalIndex(snapshot.Simulation.Portals, id)
			if !ok {
				return ErrPortalNotFound
			}
			planeAt, ok := planeIndex(snapshot.Simulation.Planes, snapshot.Simulation.Portals[portalAt].DestinationPlaneID)
			if !ok {
				return ErrPlaneNotFound
			}
			return domain.ClosePortalWithLabEnergy(
				&snapshot.Simulation.Lab,
				&snapshot.Simulation.Portals[portalAt],
				&snapshot.Simulation.Planes[planeAt],
				snapshot.Simulation.Observers,
				now,
				confirm,
				m.cfg,
			)
		},
	})
}

func (m *LabManager) SendObserver(ctx context.Context, id int64, confirm bool) error {
	return m.executeCommand(ctx, managerCommand{
		action:   "SEND",
		portalID: &id,
		apply: func(snapshot *persistence.Snapshot, now time.Time) error {
			portalAt, ok := portalIndex(snapshot.Simulation.Portals, id)
			if !ok {
				return ErrPortalNotFound
			}
			planeAt, ok := planeIndex(snapshot.Simulation.Planes, snapshot.Simulation.Portals[portalAt].DestinationPlaneID)
			if !ok {
				return ErrPlaneNotFound
			}
			_, err := domain.SendObserverWithLabEnergy(
				&snapshot.Simulation.Lab,
				&snapshot.Simulation.Portals[portalAt],
				&snapshot.Simulation.Planes[planeAt],
				snapshot.Simulation.Observers,
				now,
				confirm,
				m.random,
				m.cfg,
			)
			return err
		},
	})
}

func (m *LabManager) RecallObserver(ctx context.Context, id int64, confirm bool) error {
	return m.executeCommand(ctx, managerCommand{
		action:   "RECALL",
		portalID: &id,
		apply: func(snapshot *persistence.Snapshot, now time.Time) error {
			portalAt, ok := portalIndex(snapshot.Simulation.Portals, id)
			if !ok {
				return ErrPortalNotFound
			}
			planeAt, ok := planeIndex(snapshot.Simulation.Planes, snapshot.Simulation.Portals[portalAt].DestinationPlaneID)
			if !ok {
				return ErrPlaneNotFound
			}
			_, err := domain.RecallObserverWithLabEnergy(
				&snapshot.Simulation.Lab,
				&snapshot.Simulation.Portals[portalAt],
				&snapshot.Simulation.Planes[planeAt],
				snapshot.Simulation.Observers,
				now,
				confirm,
				m.random,
				m.cfg,
			)
			return err
		},
	})
}

func (m *LabManager) OpenExtraction(ctx context.Context, planeID int64) error {
	return m.executeCommand(ctx, managerCommand{
		action:  "OPEN_EXTRACTION",
		planeID: &planeID,
		apply: func(snapshot *persistence.Snapshot, now time.Time) error {
			planeAt, ok := planeIndex(snapshot.Simulation.Planes, planeID)
			if !ok {
				return ErrPlaneNotFound
			}
			_, err := domain.OpenExtractionPortal(
				&snapshot.Simulation.Lab,
				&snapshot.Simulation.Portals,
				&snapshot.Simulation.Planes[planeAt],
				snapshot.Simulation.Observers,
				snapshot.Simulation.NextPortalID,
				now,
				m.random,
				m.cfg,
			)
			if err == nil {
				snapshot.Simulation.NextPortalID++
			}
			return err
		},
	})
}

func (m *LabManager) executeCommand(ctx context.Context, command managerCommand) (err error) {
	if m == nil {
		return fmt.Errorf("%s: nil manager", command.action)
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
	if !reflect.DeepEqual(resolvedBeforeTutorial.Simulation, working.Simulation) {
		tutorialDrafts, eventErr := domain.EventsForStateTransition(
			resolvedBeforeTutorial.Simulation, working.Simulation, now, now, m.cfg,
		)
		if eventErr != nil {
			return eventErr
		}
		tick.Events = append(tick.Events, tutorialDrafts...)
	}
	resolved := cloneSnapshot(working)
	commandErr := command.apply(&working, now)
	if err := m.advanceTutorialAfterCommand(&working, &resolved, command, commandErr, now); err != nil {
		return err
	}
	drafts := cloneDrafts(tick.Events)
	if commandErr == nil {
		commandDrafts, eventErr := domain.EventsForStateTransition(
			resolved.Simulation, working.Simulation, now, now, m.cfg,
		)
		if eventErr != nil {
			return eventErr
		}
		drafts = append(drafts, commandDrafts...)
	} else {
		rejected, eventErr := domain.NewActionRejectedEvent(
			now,
			command.action,
			command.portalID,
			command.observerID,
			command.planeID,
			commandErr,
		)
		if eventErr != nil {
			return eventErr
		}
		drafts = append(drafts, rejected)
	}

	if _, err := m.repo.Commit(ctx, cloneSnapshot(working), cloneDrafts(drafts)); err != nil {
		return fmt.Errorf("commit %s: %w", command.action, err)
	}
	m.snapshot = cloneSnapshot(working)
	randomTx.commit()
	m.signalLocked()
	return commandErr
}
