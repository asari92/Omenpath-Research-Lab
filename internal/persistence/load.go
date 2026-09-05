package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
)

func (s *LabRepository) Load(ctx context.Context) (Snapshot, error) {
	if !s.valid() {
		return Snapshot{}, fmt.Errorf("load: nil store")
	}
	tx, err := s.store.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return Snapshot{}, fmt.Errorf("begin snapshot load: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var snapshot Snapshot
	var energyAt int64
	var overrideUntil sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT energy_base, energy_base_at, override_until
		FROM lab_state WHERE lab_id = ?`, s.labID).Scan(
		&snapshot.Simulation.Lab.EnergyBase, &energyAt, &overrideUntil,
	); err != nil {
		return Snapshot{}, fmt.Errorf("load lab state: %w", err)
	}
	snapshot.Simulation.Lab.EnergyBaseAt = decodeTime(energyAt)
	snapshot.Simulation.Lab.LeylineOverrideUntil = decodeOptionalTime(overrideUntil)

	var mode string
	var scheduledAt, dueAt, lastTickAt sql.NullInt64
	var phase string
	var tutorialPortalID, tutorialPlaneID, tutorialObserverID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT mode, tutorial_step, tutorial_phase,
		tutorial_portal_id, tutorial_plane_id, tutorial_observer_id, next_portal_id,
		spawn_scheduled_at, spawn_due_at, spawn_paused, last_tick_at
		FROM app_state WHERE lab_id = ?`, s.labID).Scan(
		&mode, &snapshot.App.TutorialStep, &phase,
		&tutorialPortalID, &tutorialPlaneID, &tutorialObserverID, &snapshot.Simulation.NextPortalID,
		&scheduledAt, &dueAt, &snapshot.Simulation.NaturalSpawn.Paused, &lastTickAt,
	); err != nil {
		return Snapshot{}, fmt.Errorf("load app state: %w", err)
	}
	snapshot.App.Mode = domain.AppMode(mode)
	snapshot.App.TutorialPhase = domain.TutorialPhase(phase)
	snapshot.App.TutorialPortalID = decodeOptionalInt64(tutorialPortalID)
	snapshot.App.TutorialPlaneID = decodeOptionalInt64(tutorialPlaneID)
	snapshot.App.TutorialObserverID = decodeOptionalInt64(tutorialObserverID)
	snapshot.Simulation.NaturalSpawn.ScheduledAt = decodeOptionalTime(scheduledAt)
	snapshot.Simulation.NaturalSpawn.DueAt = decodeOptionalTime(dueAt)
	snapshot.Simulation.LastTickAt = decodeOptionalTime(lastTickAt)

	planes, err := loadPlanes(ctx, tx, s.labID)
	if err != nil {
		return Snapshot{}, err
	}
	portals, err := loadPortals(ctx, tx, s.labID)
	if err != nil {
		return Snapshot{}, err
	}
	observers, err := loadObservers(ctx, tx, s.labID)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot.Simulation.Planes = planes
	snapshot.Simulation.Portals = portals
	snapshot.Simulation.Observers = observers
	if err := validateSnapshot(snapshot); err != nil {
		return Snapshot{}, fmt.Errorf("validate loaded snapshot: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Snapshot{}, fmt.Errorf("commit snapshot load: %w", err)
	}
	return snapshot, nil
}

type snapshotReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadPlanes(ctx context.Context, reader snapshotReader, labID LabID) ([]domain.Plane, error) {
	rows, err := reader.QueryContext(ctx, `SELECT id, name, aliases_json, catalog_tier, explored, explored_at
		FROM planes WHERE lab_id = ? ORDER BY id`, labID)
	if err != nil {
		return nil, fmt.Errorf("query planes: %w", err)
	}
	defer rows.Close()
	planes := make([]domain.Plane, 0, 85)
	for rows.Next() {
		var plane domain.Plane
		var aliases string
		var exploredAt sql.NullInt64
		if err := rows.Scan(&plane.ID, &plane.Name, &aliases, &plane.CatalogTier, &plane.Explored, &exploredAt); err != nil {
			return nil, fmt.Errorf("scan plane: %w", err)
		}
		plane.Aliases, err = decodeAliases(aliases)
		if err != nil {
			return nil, fmt.Errorf("decode plane %d: %w", plane.ID, err)
		}
		plane.ExploredAt = decodeOptionalTime(exploredAt)
		if err := validatePlane(plane); err != nil {
			return nil, err
		}
		planes = append(planes, plane)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate planes: %w", err)
	}
	return planes, nil
}

func loadPortals(ctx context.Context, reader snapshotReader, labID LabID) ([]domain.Portal, error) {
	rows, err := reader.QueryContext(ctx, `SELECT id, name, slot_index, kind, destination_plane_id,
		energy_base, energy_base_at, energy_decay_rate, stability, opened_at,
		scheduled_close_at, instability_collapse_at, creatures_initial, observer_flow,
		extraction_synchronized_at, status, termination_reason, closed_at, created_at, updated_at
		FROM portals WHERE lab_id = ? ORDER BY id`, labID)
	if err != nil {
		return nil, fmt.Errorf("query portals: %w", err)
	}
	defer rows.Close()
	portals := make([]domain.Portal, 0)
	for rows.Next() {
		var portal domain.Portal
		var kind, stability, flow, status, reason string
		var energyAt, openedAt, scheduledCloseAt, createdAt, updatedAt int64
		var instabilityAt, extractionAt, closedAt sql.NullInt64
		if err := rows.Scan(
			&portal.ID, &portal.Name, &portal.SlotIndex, &kind, &portal.DestinationPlaneID,
			&portal.EnergyBase, &energyAt, &portal.EnergyDecayRate, &stability, &openedAt,
			&scheduledCloseAt, &instabilityAt, &portal.CreaturesInitial, &flow,
			&extractionAt, &status, &reason, &closedAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan portal: %w", err)
		}
		portal.Kind = domain.PortalKind(kind)
		portal.Stability = domain.PortalStability(stability)
		portal.ObserverFlow = domain.PortalFlow(flow)
		portal.Status = domain.PortalStatus(status)
		portal.TerminationReason = domain.TerminationReason(reason)
		portal.EnergyBaseAt = decodeTime(energyAt)
		portal.OpenedAt = decodeTime(openedAt)
		portal.ScheduledCloseAt = decodeTime(scheduledCloseAt)
		portal.InstabilityCollapseAt = decodeOptionalTime(instabilityAt)
		portal.ExtractionSynchronizedAt = decodeOptionalTime(extractionAt)
		portal.ClosedAt = decodeOptionalTime(closedAt)
		portal.CreatedAt = decodeTime(createdAt)
		portal.UpdatedAt = decodeTime(updatedAt)
		if err := validatePortal(portal); err != nil {
			return nil, err
		}
		portals = append(portals, portal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate portals: %w", err)
	}
	return portals, nil
}

func loadObservers(ctx context.Context, reader snapshotReader, labID LabID) ([]domain.Observer, error) {
	rows, err := reader.QueryContext(ctx, `SELECT id, status, current_plane_id, active_portal_id,
		phase_started_at, phase_ends_at, created_at, updated_at FROM observers WHERE lab_id = ? ORDER BY id`, labID)
	if err != nil {
		return nil, fmt.Errorf("query observers: %w", err)
	}
	defer rows.Close()
	observers := make([]domain.Observer, 0, config.Default().ObserverCount)
	for rows.Next() {
		var observer domain.Observer
		var status string
		var currentPlaneID, activePortalID, phaseStartedAt, phaseEndsAt sql.NullInt64
		var createdAt, updatedAt int64
		if err := rows.Scan(
			&observer.ID, &status, &currentPlaneID, &activePortalID,
			&phaseStartedAt, &phaseEndsAt, &createdAt, &updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan observer: %w", err)
		}
		observer.Status = domain.ObserverStatus(status)
		observer.CurrentPlaneID = decodeOptionalInt64(currentPlaneID)
		observer.ActivePortalID = decodeOptionalInt64(activePortalID)
		observer.PhaseStartedAt = decodeOptionalTime(phaseStartedAt)
		observer.PhaseEndsAt = decodeOptionalTime(phaseEndsAt)
		observer.CreatedAt = decodeTime(createdAt)
		observer.UpdatedAt = decodeTime(updatedAt)
		if err := validateObserver(observer); err != nil {
			return nil, err
		}
		observers = append(observers, observer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observers: %w", err)
	}
	return observers, nil
}

func (s *LabRepository) ListEvents(ctx context.Context, portalID *int64) ([]domain.Event, error) {
	if !s.valid() {
		return nil, fmt.Errorf("list events: nil store")
	}
	query := `SELECT id, event_type, portal_id, observer_id, plane_id, message, payload_json, created_at
		FROM events WHERE lab_id = ?`
	args := []any{s.labID}
	if portalID != nil {
		if *portalID <= 0 {
			return nil, fmt.Errorf("list events: invalid portal ID")
		}
		query += ` AND portal_id = ?`
		args = append(args, *portalID)
	}
	query += ` ORDER BY created_at ASC, id ASC`
	rows, err := s.store.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()
	events := make([]domain.Event, 0)
	for rows.Next() {
		var event domain.Event
		var eventType string
		var portal, observer, plane sql.NullInt64
		var createdAt int64
		if err := rows.Scan(
			&event.ID, &eventType, &portal, &observer, &plane,
			&event.Message, &event.PayloadJSON, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		event.EventType = domain.EventType(eventType)
		event.PortalID = decodeOptionalInt64(portal)
		event.ObserverID = decodeOptionalInt64(observer)
		event.PlaneID = decodeOptionalInt64(plane)
		event.CreatedAt = decodeTime(createdAt)
		if err := event.ValidatePersisted(); err != nil {
			return nil, fmt.Errorf("invalid persisted event %d: %w", event.ID, err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}
	return events, nil
}
