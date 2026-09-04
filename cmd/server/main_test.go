package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/httpapi"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/random"
	"omenpath-lab/internal/transport"
)

func TestServerConfigFromEnv_UsesSafeDefaultsAndOverrides(t *testing.T) {
	defaults := serverConfigFromEnv(func(string) string { return "" })
	require.Equal(t, "./omenpath.db", defaults.databasePath)
	require.Equal(t, ":8080", defaults.address)
	overrides := map[string]string{"OMENPATH_DB_PATH": "/tmp/lab.sqlite", "OMENPATH_ADDR": "127.0.0.1:9090"}
	got := serverConfigFromEnv(func(key string) string { return overrides[key] })
	require.Equal(t, "/tmp/lab.sqlite", got.databasePath)
	require.Equal(t, "127.0.0.1:9090", got.address)
}

func TestCompositionSmoke_SQLiteRESTAndWebSocketInitialSnapshot(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	store, err := persistence.Open(ctx, t.TempDir()+"/smoke.sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	require.NoError(t, store.Bootstrap(ctx, time.Now().UTC(), cfg))
	manager, err := engine.NewLabManager(ctx, cfg, clock.RealClock{}, random.NewRealRandom(), store)
	require.NoError(t, err)
	router, err := httpapi.NewRouter(manager, cfg)
	require.NoError(t, err)
	t.Cleanup(router.Close)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/state")
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.NoError(t, response.Body.Close())

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/lab"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var snapshot transport.StateSnapshot
	require.NoError(t, wsjson.Read(readCtx, conn, &snapshot))
	require.Equal(t, domain.ModeTutorial, snapshot.App.Mode)
	require.Equal(t, domain.TutorialActionCompleteIntro, *snapshot.App.ExpectedAction)
	require.Len(t, snapshot.Slots, 7)
}
