package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
)

func eventDraft(at time.Time, eventType domain.EventType, portalID *int64) domain.EventDraft {
	return domain.EventDraft{
		EventType:   eventType,
		PortalID:    portalID,
		Message:     string(eventType),
		PayloadJSON: `{}`,
		CreatedAt:   at,
	}
}

func TestStoreCommit_AssignsEventIDsInChronologicalOrder(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 0, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portalID := int64(7)
	drafts := []domain.EventDraft{
		eventDraft(now.Add(2*time.Second), domain.EventPortalStabilized, &portalID),
		eventDraft(now.Add(time.Second), domain.EventPortalOpened, &portalID),
		eventDraft(now.Add(2*time.Second), domain.EventRiskLevelChanged, &portalID),
	}

	events, err := store.Commit(ctx, snapshot, drafts)
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, []domain.EventType{
		domain.EventPortalOpened,
		domain.EventPortalStabilized,
		domain.EventRiskLevelChanged,
	}, []domain.EventType{events[0].EventType, events[1].EventType, events[2].EventType})
	require.Less(t, events[0].ID, events[1].ID)
	require.Less(t, events[1].ID, events[2].ID)
}

func TestStoreListEvents_GlobalAndPortalUseSameRows(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 10, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portal7, portal8 := int64(7), int64(8)
	_, err := store.Commit(ctx, snapshot, []domain.EventDraft{
		eventDraft(now, domain.EventPortalOpened, &portal7),
		eventDraft(now.Add(time.Second), domain.EventPortalClosed, &portal8),
		eventDraft(now.Add(2*time.Second), domain.EventPortalStabilized, &portal7),
	})
	require.NoError(t, err)

	global, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	history, err := store.ListEvents(ctx, &portal7)
	require.NoError(t, err)
	require.Equal(t, []domain.Event{global[0], global[2]}, history)
}

func TestStoreCommit_RollsBackStateWhenEventInsertFails(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 20, 0, 0, time.UTC)
	before := completeSnapshot(now)
	_, err := store.Commit(ctx, before, nil)
	require.NoError(t, err)

	after := completeSnapshot(now.Add(time.Second))
	after.Simulation.Lab.EnergyBase = 3
	_, err = store.db.Exec(`CREATE TRIGGER force_event_insert_failure
		BEFORE INSERT ON events
		WHEN NEW.message = 'force event failure'
		BEGIN
			SELECT RAISE(ABORT, 'forced event failure');
		END`)
	require.NoError(t, err)
	failingDraft := eventDraft(now.Add(time.Second), domain.EventPortalClosed, nil)
	failingDraft.Message = "force event failure"
	_, err = store.Commit(ctx, after, []domain.EventDraft{
		failingDraft,
	})
	require.Error(t, err)

	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, before, got)
	events, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestStoreCommit_ActionRejectedPreservesMissingRequestedEntityIDs(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 25, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	missingPortalID, missingObserverID, missingPlaneID := int64(999), int64(998), int64(997)
	draft := eventDraft(now, domain.EventActionRejected, &missingPortalID)
	draft.ObserverID = &missingObserverID
	draft.PlaneID = &missingPlaneID

	persisted, err := store.Commit(ctx, snapshot, []domain.EventDraft{draft})
	require.NoError(t, err)
	require.Len(t, persisted, 1)

	events, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, persisted, events)
	require.Equal(t, &missingPortalID, events[0].PortalID)
	require.Equal(t, &missingObserverID, events[0].ObserverID)
	require.Equal(t, &missingPlaneID, events[0].PlaneID)
}

func TestStoreLoad_RejectsCorruptPersistedEnum(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 30, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	_, err := store.Commit(ctx, snapshot, nil)
	require.NoError(t, err)

	_, err = store.db.Exec(`PRAGMA ignore_check_constraints = ON`)
	require.NoError(t, err)
	_, err = store.db.Exec(`UPDATE observers SET status = 'BROKEN' WHERE id = 1`)
	require.NoError(t, err)
	_, err = store.Load(ctx)
	require.Error(t, err)
}

func TestStoreListEvents_RejectsCorruptPayloadJSON(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 40, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portalID := int64(7)
	_, err := store.Commit(ctx, snapshot, []domain.EventDraft{
		eventDraft(now, domain.EventPortalOpened, &portalID),
	})
	require.NoError(t, err)

	_, err = store.db.Exec(`PRAGMA ignore_check_constraints = ON`)
	require.NoError(t, err)
	_, err = store.db.Exec(`UPDATE events SET payload_json = '[]'`)
	require.NoError(t, err)
	_, err = store.ListEvents(ctx, nil)
	require.Error(t, err)
}

func TestStoreListEvents_RejectsCorruptRequiredTimestamp(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 13, 50, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portalID := int64(7)
	_, err := store.Commit(ctx, snapshot, []domain.EventDraft{
		eventDraft(now, domain.EventPortalOpened, &portalID),
	})
	require.NoError(t, err)

	_, err = store.db.Exec(`UPDATE events SET created_at = 'broken'`)
	require.NoError(t, err)
	_, err = store.ListEvents(ctx, nil)
	require.Error(t, err)
}

func TestStoreCommitAndListEvents_RoundTripsUnixEpochTimestamp(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 14, 0, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portalID := int64(7)
	epoch := time.Unix(0, 0).UTC()
	draft := eventDraft(epoch, domain.EventPortalOpened, &portalID)

	persisted, err := store.Commit(ctx, snapshot, []domain.EventDraft{draft})
	require.NoError(t, err)
	require.Len(t, persisted, 1)

	events, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, persisted, events)
	require.Equal(t, epoch, events[0].CreatedAt)
}
