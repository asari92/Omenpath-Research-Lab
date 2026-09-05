package persistence

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
)

func openMigratedStore(t *testing.T) *Store {
	t.Helper()

	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "lab.sqlite"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(context.Background()))
	return store
}

func testLab(t *testing.T, store *Store) *LabRepository {
	t.Helper()
	repo, err := store.ForLab(LabID(tenantA))
	require.NoError(t, err)
	return repo
}

func openBootstrappedStore(t *testing.T) *Store {
	t.Helper()
	store := openMigratedStore(t)
	require.NoError(t, testLab(t, store).Bootstrap(context.Background(), time.Now().UTC(), config.Default()))
	return store
}

func TestStoreMigrate_CreatesRequiredTablesAndIndexes(t *testing.T) {
	store := openMigratedStore(t)

	rows, err := store.db.Query(`SELECT type, name FROM sqlite_master
		WHERE (type = 'table' AND name NOT LIKE 'sqlite_%') OR type = 'index'`)
	require.NoError(t, err)
	defer rows.Close()

	var tables, indexes []string
	for rows.Next() {
		var kind, name string
		require.NoError(t, rows.Scan(&kind, &name))
		switch kind {
		case "table":
			tables = append(tables, name)
		case "index":
			indexes = append(indexes, name)
		}
	}
	require.NoError(t, rows.Err())
	sort.Strings(tables)
	sort.Strings(indexes)
	require.Equal(t,
		[]string{"app_state", "events", "lab_state", "labs", "observers", "planes", "portals", "schema_migrations", "sessions"},
		tables,
	)
	require.Subset(t, indexes, []string{
		"idx_events_created_at_id",
		"idx_events_portal_created_at_id",
		"idx_portals_one_open_per_slot",
	})

	var foreignKeys, busyTimeout int
	require.NoError(t, store.db.QueryRow(`PRAGMA foreign_keys`).Scan(&foreignKeys))
	require.NoError(t, store.db.QueryRow(`PRAGMA busy_timeout`).Scan(&busyTimeout))
	require.Equal(t, 1, foreignKeys)
	require.GreaterOrEqual(t, busyTimeout, 1)
}

func TestStoreMigrate_EventEntityIDsAreSoftReferences(t *testing.T) {
	store := openMigratedStore(t)

	rows, err := store.db.Query(`PRAGMA foreign_key_list(events)`)
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next(), "events must belong to a laboratory")
	var id, seq int
	var target, from, to, update, deleteRule, match string
	require.NoError(t, rows.Scan(&id, &seq, &target, &from, &to, &update, &deleteRule, &match))
	require.Equal(t, "labs", target)
	require.Equal(t, "lab_id", from)
	require.Equal(t, "CASCADE", deleteRule)
	require.False(t, rows.Next(), "event entity IDs must remain soft references")
	require.NoError(t, rows.Err())
}

func TestStoreMigrate_IsIdempotent(t *testing.T) {
	store := openMigratedStore(t)

	require.NoError(t, store.Migrate(context.Background()))
	require.NoError(t, store.Migrate(context.Background()))

	var count int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count))
	require.Equal(t, 1, count)
}

func TestStoreBootstrap_SeedsExactly85PlanesAnd20Observers(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 10, 11, 12, 13, time.UTC)

	require.NoError(t, testLab(t, store).Bootstrap(ctx, now, config.Default()))

	var planes, observers, available, explored, portals, events int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM planes`).Scan(&planes))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM observers`).Scan(&observers))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM observers WHERE status = 'AVAILABLE'`).Scan(&available))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM planes WHERE explored = 1`).Scan(&explored))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM portals`).Scan(&portals))
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM events`).Scan(&events))
	require.Equal(t, 85, planes)
	require.Equal(t, 20, observers)
	require.Equal(t, 20, available)
	rows, err := store.db.Query(`SELECT id, status FROM observers ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()
	for expectedID := int64(1); expectedID <= 20; expectedID++ {
		require.True(t, rows.Next())
		var id int64
		var status string
		require.NoError(t, rows.Scan(&id, &status))
		require.Equal(t, expectedID, id)
		require.Equal(t, "AVAILABLE", status)
	}
	require.False(t, rows.Next())
	require.NoError(t, rows.Err())
	require.Zero(t, explored)
	require.Zero(t, portals)
	require.Zero(t, events)
}

func TestStoreBootstrap_UsesTutorialEnergy100AndStep0(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 10, 11, 12, 13, time.UTC)

	require.NoError(t, testLab(t, store).Bootstrap(ctx, now, config.Default()))

	var energy int
	var energyAt int64
	var overrideUntil *int64
	require.NoError(t, store.db.QueryRow(
		`SELECT energy_base, energy_base_at, override_until FROM lab_state WHERE lab_id = '0123456789abcdef0123456789abcdef'`,
	).Scan(&energy, &energyAt, &overrideUntil))
	require.Equal(t, 100, energy)
	require.Equal(t, now.UnixNano(), energyAt)
	require.Nil(t, overrideUntil)

	var mode string
	var step int
	var nextPortalID int64
	var scheduledAt, dueAt, lastTickAt *int64
	var paused bool
	require.NoError(t, store.db.QueryRow(`SELECT mode, tutorial_step, next_portal_id,
		spawn_scheduled_at, spawn_due_at, spawn_paused, last_tick_at FROM app_state WHERE lab_id = '0123456789abcdef0123456789abcdef'`).Scan(
		&mode, &step, &nextPortalID, &scheduledAt, &dueAt, &paused, &lastTickAt,
	))
	require.Equal(t, "TUTORIAL", mode)
	require.Zero(t, step)
	require.EqualValues(t, 1, nextPortalID)
	require.Nil(t, scheduledAt)
	require.Nil(t, dueAt)
	require.True(t, paused)
	require.Nil(t, lastTickAt)
}

func TestStoreBootstrap_DoesNotOverwriteExistingState(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	first := time.Date(2026, 9, 4, 10, 0, 0, 0, time.UTC)
	second := first.Add(time.Hour)
	require.NoError(t, testLab(t, store).Bootstrap(ctx, first, config.Default()))

	_, err := store.db.Exec(`UPDATE lab_state SET energy_base = 37; UPDATE app_state SET tutorial_step = 4; UPDATE planes SET explored = 1, explored_at = ? WHERE id = 1`, first.UnixNano())
	require.NoError(t, err)
	require.NoError(t, testLab(t, store).Bootstrap(ctx, second, config.Default()))

	var energy, step, explored int
	var exploredAt int64
	require.NoError(t, store.db.QueryRow(`SELECT energy_base FROM lab_state WHERE lab_id = '0123456789abcdef0123456789abcdef'`).Scan(&energy))
	require.NoError(t, store.db.QueryRow(`SELECT tutorial_step FROM app_state WHERE lab_id = '0123456789abcdef0123456789abcdef'`).Scan(&step))
	require.NoError(t, store.db.QueryRow(`SELECT explored, explored_at FROM planes WHERE id = 1`).Scan(&explored, &exploredAt))
	require.Equal(t, 37, energy)
	require.Equal(t, 4, step)
	require.Equal(t, 1, explored)
	require.Equal(t, first.UnixNano(), exploredAt)
}

func TestStoreBootstrap_DoesNotDependOnWorkingDirectory(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 4, 10, 11, 12, 13, time.UTC)
	original, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(t.TempDir()))
	t.Cleanup(func() { require.NoError(t, os.Chdir(original)) })

	require.NoError(t, testLab(t, store).Bootstrap(ctx, now, config.Default()))

	var count int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM planes`).Scan(&count))
	require.Equal(t, 85, count)
}
