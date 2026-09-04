package engine

import (
	"context"
	"fmt"
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

func (m *LabManager) executeCommand(ctx context.Context, command managerCommand) error {
	if m == nil {
		return fmt.Errorf("%s: nil manager", command.action)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.clock.Now().UTC()
	working := cloneSnapshot(m.snapshot)
	tick, err := working.Simulation.ResolveTick(now, m.random, m.cfg)
	if err != nil {
		return err
	}
	resolved := cloneSnapshot(working)
	commandErr := command.apply(&working, now)
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
	m.signalLocked()
	return commandErr
}
