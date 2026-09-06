package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/labruntime"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/session"
	"omenpath-lab/internal/transport"
	"omenpath-lab/testutil"
)

// Test-owned harness composes the same SQLite, resolver, registry and router as
// cmd/server. Time and storage are controlled without any production test route.
type closureHarness struct {
	store    *persistence.Store
	registry *labruntime.Registry
	resolver *session.Service
	router   *Router
	server   *httptest.Server
}

func openClosureHarness(t *testing.T, path string, c *testutil.FakeClock) *closureHarness {
	t.Helper()
	ctx := context.Background()
	cfg := config.Default()
	store, err := persistence.Open(ctx, path)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(ctx))
	registry, err := labruntime.New(store, cfg, c)
	require.NoError(t, err)
	resolver, err := session.New(store, cfg, c, nil)
	require.NoError(t, err)
	router, err := NewRouter(resolver, registry, cfg, false)
	require.NoError(t, err)
	return &closureHarness{store: store, registry: registry, resolver: resolver, router: router, server: httptest.NewServer(router)}
}

func (h *closureHarness) close(t *testing.T) {
	t.Helper()
	h.registry.Close()
	h.server.Close()
	require.NoError(t, h.store.Close())
}

func TestSessionClosure_ServerRestartResumesExactCookieLaboratory(t *testing.T) {
	c := testutil.NewFakeClock(testutil.BaseTime)
	path := t.TempDir() + "/restart.sqlite"
	first := openClosureHarness(t, path, c)
	response := sessionRequest(t, first.router, "GET", "/api/state", "", nil)
	require.Equal(t, 200, response.Code)
	cookie := response.Result().Cookies()[0]
	resolved, err := first.resolver.Resolve(context.Background(), cookie.Value)
	require.NoError(t, err)
	require.Equal(t, 200, sessionRequest(t, first.router, "POST", "/api/tutorial/signal", `{"signal":"TUTORIAL_INTRO_COMPLETED"}`, cookie).Code)
	require.Equal(t, 200, sessionRequest(t, first.router, "POST", "/api/portals/1/close", `{"confirm":true}`, cookie).Code)
	before := sessionRequest(t, first.router, "GET", "/api/state", "", cookie)
	events := sessionRequest(t, first.router, "GET", "/api/events", "", cookie)
	first.close(t)

	second := openClosureHarness(t, path, c)
	defer second.close(t)
	resumed, err := second.resolver.Resolve(context.Background(), cookie.Value)
	require.NoError(t, err)
	require.Equal(t, resolved.LabID, resumed.LabID)
	require.False(t, resumed.Reissue)
	after := sessionRequest(t, second.router, "GET", "/api/state", "", cookie)
	require.Equal(t, 200, after.Code)
	require.Empty(t, after.Result().Cookies())
	require.JSONEq(t, before.Body.String(), after.Body.String())
	require.JSONEq(t, events.Body.String(), sessionRequest(t, second.router, "GET", "/api/events", "", cookie).Body.String())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, realtimeURL(second.server.URL), &websocket.DialOptions{HTTPHeader: http.Header{"Cookie": []string{cookie.String()}}})
	require.NoError(t, err)
	defer conn.CloseNow()
	var want transport.StateSnapshot
	require.NoError(t, json.Unmarshal(after.Body.Bytes(), &want))
	require.Equal(t, want, receiveRouterSnapshot(t, conn))
	newGame := sessionRequest(t, second.router, "GET", "/api/state", "", nil)
	require.NotEqual(t, cookie.Value, newGame.Result().Cookies()[0].Value)
	require.NotContains(t, after.Body.String(), string(resumed.LabID))
	require.NotContains(t, after.Body.String(), cookie.Value)
}

func TestSessionClosure_ExpiredLiveWebSocketSurvivesCleanupUntilDisconnect(t *testing.T) {
	ctx := context.Background()
	c := testutil.NewFakeClock(testutil.BaseTime)
	h := openClosureHarness(t, t.TempDir()+"/cleanup.sqlite", c)
	defer h.close(t)
	resolution, err := h.resolver.Resolve(ctx, "")
	require.NoError(t, err)
	cookie := &http.Cookie{Name: config.Default().SessionCookieName, Value: resolution.Token}
	dialCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, realtimeURL(h.server.URL), &websocket.DialOptions{HTTPHeader: http.Header{"Cookie": []string{cookie.String()}}})
	require.NoError(t, err)
	defer conn.CloseNow()
	_ = receiveRouterSnapshot(t, conn)
	now := c.Advance(config.Default().SessionTTL)
	candidates, err := h.store.ExpiredLabs(ctx, now)
	require.NoError(t, err)
	require.Equal(t, []persistence.LabID{resolution.LabID}, candidates)
	called := false
	idle, err := h.registry.DeleteIfIdle(resolution.LabID, func() error {
		called = true
		_, err := h.store.DeleteExpiredLab(ctx, resolution.LabID, now)
		return err
	})
	require.NoError(t, err)
	require.False(t, idle)
	require.False(t, called)
	repo, err := h.store.ForLab(resolution.LabID)
	require.NoError(t, err)
	_, err = repo.Load(ctx)
	require.NoError(t, err)
	require.NoError(t, conn.CloseNow())
	deleted := false
	require.Eventually(t, func() bool {
		idle, err = h.registry.DeleteIfIdle(resolution.LabID, func() error {
			var deleteErr error
			deleted, deleteErr = h.store.DeleteExpiredLab(ctx, resolution.LabID, now)
			return deleteErr
		})
		return err == nil && idle
	}, 3*time.Second, time.Millisecond)
	require.True(t, deleted)
	_, err = repo.Load(ctx)
	require.Error(t, err)
}

func TestSessionClosure_RenewalWinsAgainstPreviouslySelectedCleanupCandidate(t *testing.T) {
	ctx := context.Background()
	c := testutil.NewFakeClock(testutil.BaseTime)
	h := openClosureHarness(t, t.TempDir()+"/renewal.sqlite", c)
	defer h.close(t)
	resolution, err := h.resolver.Resolve(ctx, "")
	require.NoError(t, err)
	// A cleanup pass has a cutoff and candidate list. Renewal commits before
	// it obtains the idle gate; deleting must recheck the current row expiry.
	cutoff := c.Now().Add(config.Default().SessionTTL)
	candidates, err := h.store.ExpiredLabs(ctx, cutoff)
	require.NoError(t, err)
	require.Equal(t, []persistence.LabID{resolution.LabID}, candidates)
	c.Advance(config.Default().SessionRefreshInterval)
	cookie := &http.Cookie{Name: config.Default().SessionCookieName, Value: resolution.Token}
	response := sessionRequest(t, h.router, "GET", "/api/state", "", cookie)
	require.Equal(t, 200, response.Code)
	require.Equal(t, cookie.Value, response.Result().Cookies()[0].Value)
	deleted := true
	idle, err := h.registry.DeleteIfIdle(candidates[0], func() error {
		var deleteErr error
		deleted, deleteErr = h.store.DeleteExpiredLab(ctx, candidates[0], cutoff)
		return deleteErr
	})
	require.NoError(t, err)
	require.True(t, idle)
	require.False(t, deleted)
	require.Equal(t, 200, sessionRequest(t, h.router, "GET", "/api/state", "", cookie).Code)
}
