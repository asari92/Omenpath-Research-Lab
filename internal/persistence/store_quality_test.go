package persistence

import (
	"context"
	"database/sql/driver"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"modernc.org/sqlite"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
)

var loadBarrierSequence atomic.Uint64

func TestStoreCommitAndLoad_AcceptsStructurallyValidCustomTimingSnapshot(t *testing.T) {
	ctx := context.Background()
	store := openBootstrappedStore(t)
	snapshot := completeSnapshot(time.Date(2026, 9, 4, 16, 0, 0, 0, time.UTC))
	customSynchronization := snapshot.Simulation.Portals[1].OpenedAt.Add(6 * time.Second)
	snapshot.Simulation.Portals[1].ExtractionSynchronizedAt = &customSynchronization

	_, err := testLab(t, store).Commit(ctx, snapshot, nil)
	require.NoError(t, err)
	loaded, err := testLab(t, store).Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot, loaded)
}

func TestStoreCommitAndLoad_AcceptsCommandUpdatesAfterLastTick(t *testing.T) {
	ctx := context.Background()
	store := openBootstrappedStore(t)
	now := time.Date(2026, 9, 4, 16, 10, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	lastTick := now.Add(-10 * time.Second)
	snapshot.Simulation.LastTickAt = &lastTick

	_, err := testLab(t, store).Commit(ctx, snapshot, nil)
	require.NoError(t, err)
	loaded, err := testLab(t, store).Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot, loaded)
}

func TestSQLiteDSN_PreservesMemoryURIAndMergesConnectionSettings(t *testing.T) {
	name := filepath.Join(t.TempDir(), "shared memory")
	raw := "file:" + name + "?mode=memory&cache=shared&_fk=0&_timeout=1"

	dsn, err := sqliteDSN(raw)
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "file", parsed.Scheme)
	require.Equal(t, name, parsed.Path)
	require.Equal(t, "memory", parsed.Query().Get("mode"))
	require.Equal(t, "shared", parsed.Query().Get("cache"))
	require.Equal(t, "1", parsed.Query().Get("_foreign_keys"))
	require.Equal(t, "5000", parsed.Query().Get("_busy_timeout"))
	require.Empty(t, parsed.Query().Get("_fk"))
	require.Empty(t, parsed.Query().Get("_timeout"))
}

func TestStoreOpen_SharedMemoryURIWorksAcrossConnectionsAndThenVanishes(t *testing.T) {
	ctx := context.Background()
	uri := "file:" + filepath.Join(t.TempDir(), "shared-state") + "?mode=memory&cache=shared"
	first, err := Open(ctx, uri)
	require.NoError(t, err)
	require.NoError(t, first.Migrate(ctx))
	require.NoError(t, testLab(t, first).Bootstrap(ctx, time.Now().UTC(), config.Default()))
	want := completeSnapshot(time.Date(2026, 9, 4, 16, 20, 0, 0, time.UTC))
	_, err = testLab(t, first).Commit(ctx, want, nil)
	require.NoError(t, err)

	second, err := Open(ctx, uri)
	require.NoError(t, err)
	got, err := testLab(t, second).Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NoError(t, first.Close())

	third, err := Open(ctx, uri)
	require.NoError(t, err)
	got, err = testLab(t, third).Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.NoError(t, third.Close())
	require.NoError(t, second.Close())

	fresh, err := Open(ctx, uri)
	require.NoError(t, err)
	defer func() { require.NoError(t, fresh.Close()) }()
	_, err = testLab(t, fresh).Load(ctx)
	require.Error(t, err, "named memory database must disappear after its last connection closes")
}

func TestStoreOpen_AdversarialURIPragmaCannotOverrideMandatorySettings(t *testing.T) {
	ctx := context.Background()
	uri := "file:" + filepath.Join(t.TempDir(), "adversarial") +
		"?mode=memory&cache=shared&_pragma=main.busy_timeout%281%29&_pragma=foreign_keys%280%29"
	dsn, err := sqliteDSN(uri)
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "memory", parsed.Query().Get("mode"))
	require.Equal(t, "shared", parsed.Query().Get("cache"))

	store, err := Open(ctx, uri)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	store.db.SetMaxIdleConns(0)

	assertMandatorySettings := func() {
		conn, connErr := store.db.Conn(ctx)
		require.NoError(t, connErr)
		var foreignKeys, busyTimeout int
		require.NoError(t, conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys))
		require.NoError(t, conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout))
		require.Equal(t, 1, foreignKeys)
		require.Equal(t, 5000, busyTimeout)
		require.NoError(t, conn.Close())
	}

	assertMandatorySettings()
	assertMandatorySettings()
}

func TestStoreCommit_RejectsStructurallyImpossibleSnapshots(t *testing.T) {
	now := time.Date(2026, 9, 4, 15, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		mutate func(*Snapshot)
	}{
		{
			name: "available observer carries plane and phase",
			mutate: func(snapshot *Snapshot) {
				planeID := int64(1)
				phase := now.Add(-time.Second)
				snapshot.Simulation.Observers[1].CurrentPlaneID = &planeID
				snapshot.Simulation.Observers[1].PhaseStartedAt = &phase
			},
		},
		{
			name: "natural portal carries extraction synchronization",
			mutate: func(snapshot *Snapshot) {
				synchronizedAt := snapshot.Simulation.Portals[0].OpenedAt.Add(5 * time.Second)
				snapshot.Simulation.Portals[0].ExtractionSynchronizedAt = &synchronizedAt
			},
		},
		{
			name: "closed portal carries collapse reason",
			mutate: func(snapshot *Snapshot) {
				snapshot.Simulation.Portals[1].TerminationReason = domain.TerminationEnergyDepleted
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openBootstrappedStore(t)
			snapshot := completeSnapshot(now)
			test.mutate(&snapshot)

			_, err := testLab(t, store).Commit(context.Background(), snapshot, nil)
			require.Error(t, err)
		})
	}
}

func TestStoreCommit_RejectsNonFinitePortalEnergy(t *testing.T) {
	now := time.Date(2026, 9, 4, 15, 10, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*domain.Portal)
	}{
		{name: "energy NaN", mutate: func(portal *domain.Portal) { portal.EnergyBase = math.NaN() }},
		{name: "energy positive infinity", mutate: func(portal *domain.Portal) { portal.EnergyBase = math.Inf(1) }},
		{name: "energy negative infinity", mutate: func(portal *domain.Portal) { portal.EnergyBase = math.Inf(-1) }},
		{name: "decay NaN", mutate: func(portal *domain.Portal) { portal.EnergyDecayRate = math.NaN() }},
		{name: "decay positive infinity", mutate: func(portal *domain.Portal) { portal.EnergyDecayRate = math.Inf(1) }},
		{name: "decay negative infinity", mutate: func(portal *domain.Portal) { portal.EnergyDecayRate = math.Inf(-1) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := openBootstrappedStore(t)
			snapshot := completeSnapshot(now)
			test.mutate(&snapshot.Simulation.Portals[0])

			_, err := testLab(t, store).Commit(context.Background(), snapshot, nil)
			require.Error(t, err)
		})
	}
}

func TestStoreLoad_RejectsStructurallyCorruptRows(t *testing.T) {
	now := time.Date(2026, 9, 4, 15, 20, 0, 0, time.UTC)
	tests := []struct {
		name string
		sql  string
		args []any
	}{
		{
			name: "available observer carries plane and phase",
			sql:  `UPDATE observers SET current_plane_id = 1, phase_started_at = ? WHERE id = 2`,
			args: []any{now.Add(-time.Second).UnixNano()},
		},
		{
			name: "natural portal carries extraction synchronization",
			sql:  `UPDATE portals SET extraction_synchronized_at = ? WHERE id = 7`,
			args: []any{now.Add(-25 * time.Second).UnixNano()},
		},
		{
			name: "closed portal carries energy depleted reason",
			sql:  `UPDATE portals SET termination_reason = 'ENERGY_DEPLETED' WHERE id = 8`,
		},
		{
			name: "portal energy is positive infinity",
			sql:  `UPDATE portals SET energy_base = ? WHERE id = 7`,
			args: []any{math.Inf(1)},
		},
		{
			name: "portal decay is positive infinity",
			sql:  `UPDATE portals SET energy_decay_rate = ? WHERE id = 7`,
			args: []any{math.Inf(1)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			store := openBootstrappedStore(t)
			_, err := testLab(t, store).Commit(ctx, completeSnapshot(now), nil)
			require.NoError(t, err)
			_, err = store.db.ExecContext(ctx, test.sql, test.args...)
			require.NoError(t, err)

			_, err = testLab(t, store).Load(ctx)
			require.Error(t, err)
		})
	}
}

func TestStoreLoad_ReadsOneConsistentSnapshotDuringConcurrentWrite(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "consistent.sqlite")
	loadReachedApp := make(chan struct{})
	releaseLoad := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseLoad) }) }
	t.Cleanup(release)
	var blockOnce sync.Once
	barrierName := fmt.Sprintf("stage10_load_app_barrier_%d", loadBarrierSequence.Add(1))
	require.NoError(t, sqlite.RegisterScalarFunction(
		barrierName, 0,
		func(_ *sqlite.FunctionContext, _ []driver.Value) (driver.Value, error) {
			blockOnce.Do(func() {
				close(loadReachedApp)
				<-releaseLoad
			})
			return int64(1), nil
		},
	))

	store, err := Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	require.NoError(t, testLab(t, store).Bootstrap(ctx, time.Now().UTC(), config.Default()))
	before := completeSnapshot(time.Date(2026, 9, 4, 15, 30, 0, 0, time.UTC))
	_, err = testLab(t, store).Commit(ctx, before, nil)
	require.NoError(t, err)

	var journalMode string
	require.NoError(t, store.db.QueryRowContext(ctx, `PRAGMA journal_mode = WAL`).Scan(&journalMode))
	require.Equal(t, "wal", journalMode)
	_, err = store.db.ExecContext(ctx, `ALTER TABLE app_state RENAME TO app_state_rows`)
	require.NoError(t, err)
	_, err = store.db.ExecContext(ctx, fmt.Sprintf(`CREATE VIEW app_state AS
		SELECT lab_id,
			CASE %s() WHEN 1 THEN mode ELSE mode END AS mode,
			tutorial_step, tutorial_phase, tutorial_portal_id, tutorial_plane_id,
			tutorial_observer_id, next_portal_id, spawn_scheduled_at, spawn_due_at,
			spawn_paused, last_tick_at
		FROM app_state_rows`, barrierName))
	require.NoError(t, err)

	writer, err := Open(ctx, dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, writer.Close()) })

	type loadResult struct {
		snapshot Snapshot
		err      error
	}
	loaded := make(chan loadResult, 1)
	go func() {
		snapshot, loadErr := testLab(t, store).Load(ctx)
		loaded <- loadResult{snapshot: snapshot, err: loadErr}
	}()

	<-loadReachedApp
	_, err = writer.db.ExecContext(ctx, `UPDATE planes SET name = 'concurrent version' WHERE id = 1`)
	require.NoError(t, err)
	release()

	result := <-loaded
	require.NoError(t, result.err)
	require.Equal(t, before, result.snapshot)
}

func TestStoreOpen_ConfiguresEveryReplacementConnectionAndEscapesPath(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "lab ?# state.sqlite")
	store, err := Open(ctx, dbPath)
	require.NoError(t, err)
	store.db.SetMaxIdleConns(0)

	assertPragmas := func() {
		conn, connErr := store.db.Conn(ctx)
		require.NoError(t, connErr)
		var foreignKeys, busyTimeout int
		require.NoError(t, conn.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys))
		require.NoError(t, conn.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout))
		require.Equal(t, 1, foreignKeys)
		require.Equal(t, 5000, busyTimeout)
		require.NoError(t, conn.Close())
	}

	assertPragmas()
	assertPragmas()
	require.NoError(t, store.Close())
	require.FileExists(t, dbPath)
}
