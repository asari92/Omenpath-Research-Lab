package persistence

import (
	"context"
	"crypto/sha256"
	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"testing"
	"time"
)

type sessionStorage interface {
	CreateSession(context.Context, LabID, [32]byte, time.Time, config.Config) error
	ResolveSession(context.Context, [32]byte, time.Time, config.Config) (LabID, time.Time, bool, error)
	ExpiredLabs(context.Context, time.Time) ([]LabID, error)
	DeleteExpiredLab(context.Context, LabID, time.Time) (bool, error)
}

func sessionAPI(t *testing.T, s *Store) sessionStorage {
	t.Helper()
	api, ok := any(s).(sessionStorage)
	require.True(t, ok, "Store must implement atomic session lifecycle and conditional cleanup")
	return api
}

func TestSessionStore_AtomicCanonicalBootstrapAndHashOnlyStorage(t *testing.T) {
	s := openMigratedStore(t)
	api := sessionAPI(t, s)
	ctx := context.Background()
	now := time.Date(2026, 9, 6, 0, 0, 0, 7, time.UTC)
	id := LabID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash := sha256.Sum256([]byte("opaque-token"))
	require.NoError(t, api.CreateSession(ctx, id, hash, now, config.Default()))
	for table, want := range map[string]int{"labs": 1, "sessions": 1, "planes": 85, "observers": 20, "lab_state": 1, "app_state": 1} {
		var n int
		require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&n))
		require.Equal(t, want, n, table)
	}
	var got []byte
	var expiry, seen int64
	require.NoError(t, s.db.QueryRow("SELECT token_hash,expires_at,last_seen_at FROM sessions").Scan(&got, &expiry, &seen))
	require.Equal(t, hash[:], got)
	require.Equal(t, now.Add(30*24*time.Hour).UnixNano(), expiry)
	require.Equal(t, now.UnixNano(), seen)
	repo, err := s.ForLab(id)
	require.NoError(t, err)
	snap, err := repo.Load(ctx)
	require.NoError(t, err)
	require.Len(t, snap.Simulation.Observers, 20)
	require.Len(t, snap.Simulation.Planes, 85)
}

func TestSessionStore_FailuresRollbackAllRows(t *testing.T) {
	for _, table := range []string{"planes", "observers", "lab_state", "app_state", "sessions"} {
		t.Run(table, func(t *testing.T) {
			s := openMigratedStore(t)
			api := sessionAPI(t, s)
			_, err := s.db.Exec("CREATE TRIGGER fail_session BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(ABORT, 'injected failure'); END")
			require.NoError(t, err)
			require.Error(t, api.CreateSession(context.Background(), LabID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), sha256.Sum256([]byte("secret")), time.Now().UTC(), config.Default()))
			for _, name := range []string{"labs", "sessions", "planes", "observers", "lab_state", "app_state"} {
				var n int
				require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+name).Scan(&n))
				require.Zero(t, n, name)
			}
		})
	}
}

func TestSessionStore_BoundedRefreshAndConditionalCleanup(t *testing.T) {
	s := openMigratedStore(t)
	api := sessionAPI(t, s)
	ctx := context.Background()
	cfg := config.Default()
	now := time.Date(2026, 9, 6, 0, 0, 0, 0, time.UTC)
	id := LabID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	hash := sha256.Sum256([]byte("token"))
	require.NoError(t, api.CreateSession(ctx, id, hash, now, cfg))
	_, err := s.db.Exec("CREATE TRIGGER no_write BEFORE UPDATE ON sessions BEGIN SELECT RAISE(ABORT, 'unexpected write'); END")
	require.NoError(t, err)
	got, expires, refresh, err := api.ResolveSession(ctx, hash, now.Add(12*time.Hour-time.Nanosecond), cfg)
	require.NoError(t, err)
	require.Equal(t, id, got)
	require.False(t, refresh)
	require.Equal(t, now.Add(30*24*time.Hour), expires)
	_, err = s.db.Exec("DROP TRIGGER no_write")
	require.NoError(t, err)
	// Cleanup reads a candidate before a concurrent request commits its renewal.
	candidates, err := api.ExpiredLabs(ctx, now.Add(30*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, []LabID{id}, candidates)
	renewal := now.Add(12 * time.Hour)
	got, expires, refresh, err = api.ResolveSession(ctx, hash, renewal, cfg)
	require.NoError(t, err)
	require.Equal(t, id, got)
	require.True(t, refresh)
	require.Equal(t, renewal.Add(30*24*time.Hour), expires)
	var labExpiry, lastActive int64
	require.NoError(t, s.db.QueryRow("SELECT expires_at,last_active_at FROM labs WHERE id=?", id).Scan(&labExpiry, &lastActive))
	require.Equal(t, expires.UnixNano(), labExpiry)
	require.Equal(t, renewal.UnixNano(), lastActive)
	deleted, err := api.DeleteExpiredLab(ctx, id, now.Add(30*24*time.Hour))
	require.NoError(t, err)
	require.False(t, deleted)
	got, _, _, err = api.ResolveSession(ctx, hash, expires, cfg)
	require.NoError(t, err)
	require.Empty(t, got)
	deleted, err = api.DeleteExpiredLab(ctx, id, expires)
	require.NoError(t, err)
	require.True(t, deleted)
	for _, table := range []string{"labs", "sessions", "planes", "observers", "lab_state", "app_state"} {
		var n int
		require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&n))
		require.Zero(t, n, table)
	}
}
