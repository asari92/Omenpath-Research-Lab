package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
)

func (s *Store) Commit(
	ctx context.Context,
	snapshot Snapshot,
	drafts []domain.EventDraft,
) ([]domain.Event, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("commit: nil store")
	}
	if err := validateSnapshot(snapshot); err != nil {
		return nil, err
	}
	orderedDrafts := append([]domain.EventDraft(nil), drafts...)
	for i := range orderedDrafts {
		if err := orderedDrafts[i].Validate(); err != nil {
			return nil, fmt.Errorf("validate event %d: %w", i, err)
		}
	}
	sort.SliceStable(orderedDrafts, func(i, j int) bool {
		return orderedDrafts[i].CreatedAt.Before(orderedDrafts[j].CreatedAt)
	})

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin state commit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := persistSnapshot(ctx, tx, snapshot); err != nil {
		return nil, err
	}
	events := make([]domain.Event, 0, len(orderedDrafts))
	for i := range orderedDrafts {
		event, err := insertEvent(ctx, tx, orderedDrafts[i])
		if err != nil {
			return nil, fmt.Errorf("insert event %d: %w", i, err)
		}
		events = append(events, event)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit state: %w", err)
	}
	return events, nil
}

// ResetTutorial replaces all mutable state and removes prior history in one
// transaction. The supplied canonical snapshot is validated before deletion.
func (s *Store) ResetTutorial(ctx context.Context, snapshot Snapshot) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("reset tutorial: nil store")
	}
	if err := validateSnapshot(snapshot); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tutorial reset: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM events`); err != nil {
		return fmt.Errorf("clear tutorial events: %w", err)
	}
	// Active Observer rows may reference Portals. Persist the canonical
	// AVAILABLE roster first so foreign-key enforcement remains enabled.
	for _, observer := range snapshot.Simulation.Observers {
		if err := persistObserver(ctx, tx, observer); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM portals`); err != nil {
		return fmt.Errorf("clear tutorial portals: %w", err)
	}
	if err := persistSnapshot(ctx, tx, snapshot); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tutorial reset: %w", err)
	}
	return nil
}

func validateSnapshot(snapshot Snapshot) error {
	if len(snapshot.Simulation.Planes) != 85 || len(snapshot.Simulation.Observers) != config.Default().ObserverCount ||
		snapshot.Simulation.NextPortalID <= 0 || !validAppMode(snapshot.App.Mode) || snapshot.App.TutorialStep < 0 ||
		snapshot.Simulation.Lab.EnergyBase < 0 || snapshot.Simulation.Lab.EnergyBase > 100 {
		return fmt.Errorf("invalid snapshot")
	}
	if !validTutorialContext(snapshot.App) {
		return fmt.Errorf("invalid tutorial context")
	}
	if _, err := encodeTime(snapshot.Simulation.Lab.EnergyBaseAt, "lab.energy_base_at"); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}
	if _, err := encodeOptionalTime(snapshot.Simulation.Lab.LeylineOverrideUntil, "lab.override_until"); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}
	if _, err := encodeOptionalTime(snapshot.Simulation.NaturalSpawn.ScheduledAt, "app.spawn_scheduled_at"); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}
	if _, err := encodeOptionalTime(snapshot.Simulation.NaturalSpawn.DueAt, "app.spawn_due_at"); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}
	if _, err := encodeOptionalTime(snapshot.Simulation.LastTickAt, "app.last_tick_at"); err != nil {
		return fmt.Errorf("invalid snapshot: %w", err)
	}
	spawn := snapshot.Simulation.NaturalSpawn
	if spawn.Paused != (spawn.ScheduledAt == nil && spawn.DueAt == nil) ||
		(!spawn.Paused && (spawn.ScheduledAt == nil || spawn.DueAt == nil)) {
		return fmt.Errorf("invalid natural spawn state")
	}

	planeIDs := make(map[int64]struct{}, len(snapshot.Simulation.Planes))
	for _, plane := range snapshot.Simulation.Planes {
		if err := validatePlane(plane); err != nil {
			return err
		}
		if _, exists := planeIDs[plane.ID]; exists {
			return fmt.Errorf("duplicate plane %d", plane.ID)
		}
		planeIDs[plane.ID] = struct{}{}
	}
	portalIDs := make(map[int64]struct{}, len(snapshot.Simulation.Portals))
	maxPortalID := int64(0)
	for _, portal := range snapshot.Simulation.Portals {
		if err := validatePortal(portal); err != nil {
			return err
		}
		if _, exists := planeIDs[portal.DestinationPlaneID]; !exists {
			return fmt.Errorf("portal %d references missing plane", portal.ID)
		}
		if _, exists := portalIDs[portal.ID]; exists {
			return fmt.Errorf("duplicate portal %d", portal.ID)
		}
		portalIDs[portal.ID] = struct{}{}
		if portal.ID > maxPortalID {
			maxPortalID = portal.ID
		}
	}
	if snapshot.Simulation.NextPortalID <= maxPortalID {
		return fmt.Errorf("next portal ID is not ahead of persisted portals")
	}
	observerIDs := make(map[int64]struct{}, len(snapshot.Simulation.Observers))
	for _, observer := range snapshot.Simulation.Observers {
		if err := validateObserver(observer); err != nil {
			return err
		}
		if _, exists := observerIDs[observer.ID]; exists {
			return fmt.Errorf("duplicate observer %d", observer.ID)
		}
		observerIDs[observer.ID] = struct{}{}
		if observer.CurrentPlaneID != nil {
			if _, exists := planeIDs[*observer.CurrentPlaneID]; !exists {
				return fmt.Errorf("observer %d references missing plane", observer.ID)
			}
		}
		if observer.ActivePortalID != nil {
			if _, exists := portalIDs[*observer.ActivePortalID]; !exists {
				return fmt.Errorf("observer %d references missing portal", observer.ID)
			}
		}
	}
	if err := domain.ValidateSimulationStructure(
		snapshot.Simulation,
		snapshotValidationTime(snapshot.Simulation),
	); err != nil {
		return fmt.Errorf("invalid simulation snapshot: %w", err)
	}
	return nil
}

func snapshotValidationTime(state domain.SimulationState) time.Time {
	latest := state.Lab.EnergyBaseAt
	advance := func(candidate time.Time) {
		if candidate.After(latest) {
			latest = candidate
		}
	}
	if state.NaturalSpawn.ScheduledAt != nil {
		advance(*state.NaturalSpawn.ScheduledAt)
	}
	if state.LastTickAt != nil {
		advance(*state.LastTickAt)
	}
	for i := range state.Planes {
		if state.Planes[i].ExploredAt != nil {
			advance(*state.Planes[i].ExploredAt)
		}
	}
	for i := range state.Portals {
		advance(state.Portals[i].UpdatedAt)
	}
	for i := range state.Observers {
		advance(state.Observers[i].UpdatedAt)
	}
	return latest
}

func persistSnapshot(ctx context.Context, tx *sql.Tx, snapshot Snapshot) error {
	for _, plane := range snapshot.Simulation.Planes {
		aliases, err := encodeAliases(plane.Aliases)
		if err != nil {
			return err
		}
		exploredAt, err := encodeOptionalTime(plane.ExploredAt, "plane.explored_at")
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO planes
			(id, name, aliases_json, catalog_tier, explored, explored_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET name=excluded.name, aliases_json=excluded.aliases_json,
			catalog_tier=excluded.catalog_tier, explored=excluded.explored, explored_at=excluded.explored_at`,
			plane.ID, plane.Name, aliases, plane.CatalogTier, plane.Explored, exploredAt,
		); err != nil {
			return fmt.Errorf("persist plane %d: %w", plane.ID, err)
		}
	}
	for _, portal := range snapshot.Simulation.Portals {
		if err := persistPortal(ctx, tx, portal); err != nil {
			return err
		}
	}
	for _, observer := range snapshot.Simulation.Observers {
		if err := persistObserver(ctx, tx, observer); err != nil {
			return err
		}
	}
	labEnergyAt, _ := encodeTime(snapshot.Simulation.Lab.EnergyBaseAt, "lab.energy_base_at")
	overrideUntil, _ := encodeOptionalTime(snapshot.Simulation.Lab.LeylineOverrideUntil, "lab.override_until")
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_state(id, energy_base, energy_base_at, override_until)
		VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET energy_base=excluded.energy_base,
		energy_base_at=excluded.energy_base_at, override_until=excluded.override_until`,
		snapshot.Simulation.Lab.EnergyBase, labEnergyAt, overrideUntil,
	); err != nil {
		return fmt.Errorf("persist lab state: %w", err)
	}
	scheduledAt, _ := encodeOptionalTime(snapshot.Simulation.NaturalSpawn.ScheduledAt, "app.spawn_scheduled_at")
	dueAt, _ := encodeOptionalTime(snapshot.Simulation.NaturalSpawn.DueAt, "app.spawn_due_at")
	lastTickAt, _ := encodeOptionalTime(snapshot.Simulation.LastTickAt, "app.last_tick_at")
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_state
		(id, mode, tutorial_step, tutorial_phase, tutorial_portal_id, tutorial_plane_id,
		 tutorial_observer_id, next_portal_id, spawn_scheduled_at, spawn_due_at, spawn_paused, last_tick_at)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET mode=excluded.mode, tutorial_step=excluded.tutorial_step,
		tutorial_phase=excluded.tutorial_phase, tutorial_portal_id=excluded.tutorial_portal_id,
		tutorial_plane_id=excluded.tutorial_plane_id, tutorial_observer_id=excluded.tutorial_observer_id,
		next_portal_id=excluded.next_portal_id, spawn_scheduled_at=excluded.spawn_scheduled_at,
		spawn_due_at=excluded.spawn_due_at, spawn_paused=excluded.spawn_paused,
		last_tick_at=excluded.last_tick_at`,
		snapshot.App.Mode, snapshot.App.TutorialStep, snapshot.App.TutorialPhase,
		snapshot.App.TutorialPortalID, snapshot.App.TutorialPlaneID, snapshot.App.TutorialObserverID,
		snapshot.Simulation.NextPortalID,
		scheduledAt, dueAt, snapshot.Simulation.NaturalSpawn.Paused, lastTickAt,
	); err != nil {
		return fmt.Errorf("persist app state: %w", err)
	}
	return nil
}

func validTutorialContext(app domain.AppState) bool {
	if app.TutorialStep > 9 || !validOptionalID(app.TutorialPortalID) ||
		!validOptionalID(app.TutorialPlaneID) || !validOptionalID(app.TutorialObserverID) {
		return false
	}
	switch app.TutorialPhase {
	case domain.TutorialPhaseNone, domain.TutorialPhaseSendReplacement,
		domain.TutorialPhaseWaitResearch, domain.TutorialPhaseRecallReady:
		return true
	default:
		return false
	}
}

func validOptionalID(id *int64) bool { return id == nil || *id > 0 }

func persistPortal(ctx context.Context, tx *sql.Tx, portal domain.Portal) error {
	energyAt, _ := encodeTime(portal.EnergyBaseAt, "portal.energy_base_at")
	openedAt, _ := encodeTime(portal.OpenedAt, "portal.opened_at")
	scheduledCloseAt, _ := encodeTime(portal.ScheduledCloseAt, "portal.scheduled_close_at")
	instabilityAt, _ := encodeOptionalTime(portal.InstabilityCollapseAt, "portal.instability_collapse_at")
	extractionAt, _ := encodeOptionalTime(portal.ExtractionSynchronizedAt, "portal.extraction_synchronized_at")
	closedAt, _ := encodeOptionalTime(portal.ClosedAt, "portal.closed_at")
	createdAt, _ := encodeTime(portal.CreatedAt, "portal.created_at")
	updatedAt, _ := encodeTime(portal.UpdatedAt, "portal.updated_at")
	_, err := tx.ExecContext(ctx, `INSERT INTO portals
		(id, name, slot_index, kind, destination_plane_id, energy_base, energy_base_at,
		 energy_decay_rate, stability, opened_at, scheduled_close_at, instability_collapse_at,
		 creatures_initial, observer_flow, extraction_synchronized_at, status,
		 termination_reason, closed_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, slot_index=excluded.slot_index,
		kind=excluded.kind, destination_plane_id=excluded.destination_plane_id,
		energy_base=excluded.energy_base, energy_base_at=excluded.energy_base_at,
		energy_decay_rate=excluded.energy_decay_rate, stability=excluded.stability,
		opened_at=excluded.opened_at, scheduled_close_at=excluded.scheduled_close_at,
		instability_collapse_at=excluded.instability_collapse_at,
		creatures_initial=excluded.creatures_initial, observer_flow=excluded.observer_flow,
		extraction_synchronized_at=excluded.extraction_synchronized_at, status=excluded.status,
		termination_reason=excluded.termination_reason, closed_at=excluded.closed_at,
		created_at=excluded.created_at, updated_at=excluded.updated_at`,
		portal.ID, portal.Name, portal.SlotIndex, portal.Kind, portal.DestinationPlaneID,
		portal.EnergyBase, energyAt, portal.EnergyDecayRate, portal.Stability, openedAt,
		scheduledCloseAt, instabilityAt, portal.CreaturesInitial, portal.ObserverFlow,
		extractionAt, portal.Status, portal.TerminationReason, closedAt, createdAt, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("persist portal %d: %w", portal.ID, err)
	}
	return nil
}

func persistObserver(ctx context.Context, tx *sql.Tx, observer domain.Observer) error {
	phaseStartedAt, _ := encodeOptionalTime(observer.PhaseStartedAt, "observer.phase_started_at")
	phaseEndsAt, _ := encodeOptionalTime(observer.PhaseEndsAt, "observer.phase_ends_at")
	createdAt, _ := encodeTime(observer.CreatedAt, "observer.created_at")
	updatedAt, _ := encodeTime(observer.UpdatedAt, "observer.updated_at")
	_, err := tx.ExecContext(ctx, `INSERT INTO observers
		(id, status, current_plane_id, active_portal_id, phase_started_at, phase_ends_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET status=excluded.status,
		current_plane_id=excluded.current_plane_id, active_portal_id=excluded.active_portal_id,
		phase_started_at=excluded.phase_started_at, phase_ends_at=excluded.phase_ends_at,
		created_at=excluded.created_at, updated_at=excluded.updated_at`,
		observer.ID, observer.Status, encodeOptionalInt64(observer.CurrentPlaneID),
		encodeOptionalInt64(observer.ActivePortalID), phaseStartedAt, phaseEndsAt, createdAt, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("persist observer %d: %w", observer.ID, err)
	}
	return nil
}

func insertEvent(ctx context.Context, tx *sql.Tx, draft domain.EventDraft) (domain.Event, error) {
	createdAt, _ := encodeTime(draft.CreatedAt, "event.created_at")
	result, err := tx.ExecContext(ctx, `INSERT INTO events
		(event_type, portal_id, observer_id, plane_id, message, payload_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, draft.EventType, encodeOptionalInt64(draft.PortalID),
		encodeOptionalInt64(draft.ObserverID), encodeOptionalInt64(draft.PlaneID),
		draft.Message, draft.PayloadJSON, createdAt)
	if err != nil {
		return domain.Event{}, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return domain.Event{}, err
	}
	return domain.Event{
		ID:          id,
		EventType:   draft.EventType,
		PortalID:    cloneInt64(draft.PortalID),
		ObserverID:  cloneInt64(draft.ObserverID),
		PlaneID:     cloneInt64(draft.PlaneID),
		Message:     draft.Message,
		PayloadJSON: draft.PayloadJSON,
		CreatedAt:   draft.CreatedAt,
	}, nil
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
