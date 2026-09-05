package persistence

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
)

const tenantA = "0123456789abcdef0123456789abcdef"
const tenantB = "fedcba9876543210fedcba9876543210"

type tenantRepository interface {
	Bootstrap(context.Context, time.Time, config.Config) error
	Load(context.Context) (Snapshot, error)
	Commit(context.Context, Snapshot, []domain.EventDraft) ([]domain.Event, error)
	ListEvents(context.Context, *int64) ([]domain.Event, error)
	ResetTutorial(context.Context, Snapshot) error
}

// Reflection lets the schema and API contract fail behaviorally before the new
// named LabID/API exists, rather than preventing every package test compiling.
func tenantBinding(t *testing.T, store *Store, id string) (tenantRepository, error) {
	t.Helper()
	method := reflect.ValueOf(store).MethodByName("ForLab")
	require.True(t, method.IsValid(), "Store.ForLab tenant binding is missing")
	require.Equal(t, 1, method.Type().NumIn())
	require.Equal(t, "LabID", method.Type().In(0).Name())
	require.Equal(t, reflect.String, method.Type().In(0).Kind())
	result := method.Call([]reflect.Value{reflect.ValueOf(id).Convert(method.Type().In(0))})
	if !result[1].IsNil() {
		return nil, result[1].Interface().(error)
	}
	repo, ok := result[0].Interface().(tenantRepository)
	require.True(t, ok, "bound repository must implement gameplay and bootstrap")
	return repo, nil
}

func TestTenantSchema_FreshKeysForeignKeysIndexesAndEmptyState(t *testing.T) {
	store := openMigratedStore(t)
	for _, table := range []string{"labs", "sessions", "planes", "portals", "observers", "events", "lab_state", "app_state"} {
		t.Run(table, func(t *testing.T) {
			var count int
			require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
			require.Zero(t, count, "migration must not seed laboratory state")
			rows, err := store.db.Query(`PRAGMA table_info(` + table + `)`)
			require.NoError(t, err)
			keys := map[int]string{}
			columns := map[string]bool{}
			types := map[string]string{}
			requiredColumns := map[string]int{}
			for rows.Next() {
				var cid, required, pk int
				var name, typ string
				var defaultValue any
				require.NoError(t, rows.Scan(&cid, &name, &typ, &required, &defaultValue, &pk))
				columns[name] = true
				types[name] = typ
				requiredColumns[name] = required
				if pk > 0 {
					keys[pk] = name
				}
				if name == "lab_id" {
					require.Equal(t, 1, required)
				}
			}
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())
			if table == "labs" || table == "sessions" {
				wantTypes := map[string]string{"id": "TEXT", "created_at": "INTEGER", "expires_at": "INTEGER", "last_active_at": "INTEGER"}
				if table == "sessions" {
					wantTypes = map[string]string{"id": "INTEGER", "token_hash": "BLOB", "lab_id": "TEXT", "expires_at": "INTEGER", "last_seen_at": "INTEGER"}
				}
				require.Equal(t, wantTypes, types)
				for name := range wantTypes {
					if name != "id" {
						require.Equal(t, 1, requiredColumns[name], name)
					}
				}
			}
			switch table {
			case "labs", "sessions":
				require.Equal(t, map[int]string{1: "id"}, keys)
			case "lab_state", "app_state":
				require.Equal(t, map[int]string{1: "lab_id"}, keys)
				require.False(t, columns["id"])
			default:
				require.Equal(t, map[int]string{1: "lab_id", 2: "id"}, keys)
			}
			if table == "app_state" {
				for _, column := range []string{"tutorial_phase", "tutorial_portal_id", "tutorial_plane_id", "tutorial_observer_id"} {
					require.True(t, columns[column], column)
				}
			}
			rows, err = store.db.Query(`PRAGMA foreign_key_list(` + table + `)`)
			require.NoError(t, err)
			fks := map[int][]string{}
			for rows.Next() {
				var id, seq int
				var target, from, to, update, deleteRule, match string
				require.NoError(t, rows.Scan(&id, &seq, &target, &from, &to, &update, &deleteRule, &match))
				fks[id] = append(fks[id], fmt.Sprintf("%s:%s>%s:%s", target, from, to, deleteRule))
			}
			require.NoError(t, rows.Err())
			require.NoError(t, rows.Close())
			var got []string
			for _, fk := range fks {
				got = append(got, strings.Join(fk, ","))
			}
			var want []string
			if table != "labs" {
				want = append(want, "labs:lab_id>id:CASCADE")
			}
			if table == "portals" {
				want = append(want, "planes:lab_id>lab_id:NO ACTION,planes:destination_plane_id>id:NO ACTION")
			}
			if table == "observers" {
				want = append(want, "planes:lab_id>lab_id:NO ACTION,planes:current_plane_id>id:NO ACTION", "portals:lab_id>lab_id:NO ACTION,portals:active_portal_id>id:NO ACTION")
			}
			require.ElementsMatch(t, want, got)
		})
	}
	for name, columns := range map[string][]string{
		"idx_portals_one_open_per_slot":   {"lab_id", "slot_index"},
		"idx_events_created_at_id":        {"lab_id", "created_at", "id"},
		"idx_events_portal_created_at_id": {"lab_id", "portal_id", "created_at", "id"},
	} {
		rows, err := store.db.Query(`PRAGMA index_info(` + name + `)`)
		require.NoError(t, err)
		var got []string
		for rows.Next() {
			var seq, cid int
			var column string
			require.NoError(t, rows.Scan(&seq, &cid, &column))
			got = append(got, column)
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		require.Equal(t, columns, got)
	}
	var indexSQL string
	require.NoError(t, store.db.QueryRow(`SELECT sql FROM sqlite_master WHERE name='idx_portals_one_open_per_slot'`).Scan(&indexSQL))
	require.Contains(t, indexSQL, "UNIQUE INDEX")
	require.Contains(t, indexSQL, "WHERE status = 'OPEN'")
}

func TestTenantRepository_ValidatesIDAndRequiresBoundGameplay(t *testing.T) {
	store := openMigratedStore(t)
	for _, id := range []string{"", "abc", strings.Repeat("a", 31), strings.Repeat("a", 33), strings.ToUpper(tenantA), strings.Repeat("g", 32), " " + tenantA} {
		_, err := tenantBinding(t, store, id)
		require.Error(t, err, id)
	}
	for _, method := range []string{"Bootstrap", "Load", "Commit", "ListEvents", "ResetTutorial"} {
		require.False(t, reflect.ValueOf(store).MethodByName(method).IsValid(), "unbound gameplay method: %s", method)
	}
}

func TestTenantRepository_IsolatesBootstrapCommitLoadEventsResetAndRestart(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/tenants.sqlite"
	store, err := Open(ctx, path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	a, err := tenantBinding(t, store, tenantA)
	require.NoError(t, err)
	b, err := tenantBinding(t, store, tenantB)
	require.NoError(t, err)
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	for _, repo := range []tenantRepository{a, b} {
		require.NoError(t, repo.Bootstrap(ctx, now, config.Default()))
	}
	var createdAt, expiresAt, lastActiveAt int64
	require.NoError(t, store.db.QueryRow(`SELECT created_at, expires_at, last_active_at FROM labs WHERE id=?`, tenantA).Scan(&createdAt, &expiresAt, &lastActiveAt))
	require.Equal(t, now.UnixNano(), createdAt)
	require.Equal(t, now.Add(30*24*time.Hour).UnixNano(), expiresAt)
	require.Equal(t, now.UnixNano(), lastActiveAt)
	initialB, err := b.Load(ctx)
	require.NoError(t, err)
	wantA := completeSnapshot(now)
	wantB := completeSnapshot(now.Add(time.Second))
	wantB.Simulation.Lab.EnergyBase = 17
	wantB.Simulation.Planes[0].Name = "Only B"
	portalID := int64(7)
	draftA := eventDraft(now, domain.EventPortalOpened, &portalID)
	draftA.Message = "A event"
	draftB := draftA
	draftB.Message = "B event"
	eventsA, err := a.Commit(ctx, wantA, []domain.EventDraft{draftA})
	require.NoError(t, err)
	got, err := b.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, initialB, got)
	eventsB, err := b.Commit(ctx, wantB, []domain.EventDraft{draftB})
	require.NoError(t, err)
	require.EqualValues(t, 1, eventsA[0].ID)
	require.EqualValues(t, 1, eventsB[0].ID)
	for _, filter := range []*int64{nil, &portalID} {
		gotA, err := a.ListEvents(ctx, filter)
		require.NoError(t, err)
		require.Equal(t, eventsA, gotA)
		gotB, err := b.ListEvents(ctx, filter)
		require.NoError(t, err)
		require.Equal(t, eventsB, gotB)
	}
	require.NoError(t, a.Bootstrap(ctx, now.Add(time.Hour), config.Default()))
	var unchangedExpiry int64
	require.NoError(t, store.db.QueryRow(`SELECT expires_at FROM labs WHERE id=?`, tenantA).Scan(&unchangedExpiry))
	require.Equal(t, expiresAt, unchangedExpiry)
	got, err = a.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, wantA, got)
	require.NoError(t, a.ResetTutorial(ctx, resetSnapshot(now.Add(time.Minute))))
	got, err = b.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, wantB, got)
	gotEvents, err := b.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, eventsB, gotEvents)
	gotEvents, err = a.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, gotEvents)
	require.NoError(t, store.Close())
	store, err = Open(ctx, path)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(ctx))
	b, err = tenantBinding(t, store, tenantB)
	require.NoError(t, err)
	got, err = b.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, wantB, got)
}

func TestTenantBootstrap_FailureRollsBackMetadataAndChildren(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	repo := testLab(t, store)
	_, err := store.db.Exec(`CREATE TRIGGER fail_bootstrap BEFORE INSERT ON app_state
		BEGIN SELECT RAISE(ABORT, 'forced bootstrap failure'); END`)
	require.NoError(t, err)
	require.ErrorContains(t, repo.Bootstrap(ctx, time.Now().UTC(), config.Default()), "forced bootstrap failure")
	for _, table := range []string{"labs", "planes", "observers", "app_state", "lab_state"} {
		var count int
		require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM `+table).Scan(&count))
		require.Zero(t, count, table)
	}
	_, err = store.db.Exec(`DROP TRIGGER fail_bootstrap`)
	require.NoError(t, err)
	require.NoError(t, repo.Bootstrap(ctx, time.Now().UTC(), config.Default()))
}

func TestTenantSchema_RejectsCrossLabReferencesAndCascadesOnlyOneLab(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	for _, id := range []string{tenantA, tenantB} {
		repo, err := tenantBinding(t, store, id)
		require.NoError(t, err)
		require.NoError(t, repo.Bootstrap(ctx, now, config.Default()))
		_, err = repo.Commit(ctx, completeSnapshot(now), []domain.EventDraft{eventDraft(now, domain.EventPortalOpened, nil)})
		require.NoError(t, err)
	}
	_, err := store.db.Exec(`INSERT INTO planes(lab_id,id,name,aliases_json,catalog_tier,explored) VALUES (?,999,'B only','[]','core',0)`, tenantB)
	require.NoError(t, err)
	_, err = store.db.Exec(`UPDATE portals SET destination_plane_id=999 WHERE lab_id=? AND id=7`, tenantA)
	require.ErrorContains(t, err, "FOREIGN KEY")
	_, err = store.db.Exec(`UPDATE observers SET current_plane_id=999 WHERE lab_id=? AND id=1`, tenantA)
	require.ErrorContains(t, err, "FOREIGN KEY")
	_, err = store.db.Exec(`UPDATE portals SET id=999 WHERE lab_id=? AND id=8`, tenantB)
	require.NoError(t, err)
	_, err = store.db.Exec(`UPDATE observers SET active_portal_id=999 WHERE lab_id=? AND id=1`, tenantA)
	require.ErrorContains(t, err, "FOREIGN KEY")
	_, err = store.db.Exec(`UPDATE portals SET slot_index=1,status='OPEN' WHERE lab_id=? AND id=8`, tenantA)
	require.ErrorContains(t, err, "UNIQUE")
	_, err = store.db.Exec(`INSERT INTO sessions(token_hash,lab_id,expires_at,last_seen_at) VALUES (zeroblob(32),?,1,1)`, tenantA)
	require.NoError(t, err)
	for _, statement := range []string{
		`INSERT INTO sessions(token_hash,lab_id,expires_at,last_seen_at) VALUES (zeroblob(31),?,1,1)`,
		`INSERT INTO sessions(token_hash,lab_id,expires_at,last_seen_at) VALUES (zeroblob(32),?,1,1)`,
	} {
		_, err = store.db.Exec(statement, tenantB)
		require.Error(t, err)
	}
	_, err = store.db.Exec(`INSERT INTO sessions(token_hash,lab_id,expires_at,last_seen_at) VALUES (randomblob(32),?,1,1)`, tenantA)
	require.ErrorContains(t, err, "UNIQUE")
	_, err = store.db.Exec(`DELETE FROM labs WHERE id=?`, tenantA)
	require.NoError(t, err)
	for _, table := range []string{"sessions", "planes", "portals", "observers", "events", "lab_state", "app_state"} {
		var count int
		require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE lab_id=?`, tenantA).Scan(&count))
		require.Zero(t, count, table)
	}
	var count int
	require.NoError(t, store.db.QueryRow(`SELECT COUNT(*) FROM planes WHERE lab_id=?`, tenantB).Scan(&count))
	require.Equal(t, 86, count)
}
