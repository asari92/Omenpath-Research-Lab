package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/labruntime"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/session"
	"omenpath-lab/internal/transport"
)

// This fixture uses the production composition boundary; DC-4 replaces its
// singleton dependencies with session resolution and the runtime registry.
func sessionTestRouter(t *testing.T) *Router {
	t.Helper()
	ctx := context.Background()
	cfg := config.Default()
	store, err := persistence.Open(ctx, t.TempDir()+"/sessions.sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	registry, err := labruntime.New(store, cfg, clock.RealClock{})
	require.NoError(t, err)
	t.Cleanup(registry.Close)
	resolver, err := session.New(store, cfg, clock.RealClock{}, nil)
	require.NoError(t, err)
	router, err := NewRouter(resolver, registry, cfg, false)
	require.NoError(t, err)
	return router
}

func sessionRequest(t *testing.T, router http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestSessions_RESTCookieAndMatchingEntityIsolation(t *testing.T) {
	router := sessionTestRouter(t)
	a := sessionRequest(t, router, "GET", "/api/state", "", nil)
	b := sessionRequest(t, router, "GET", "/api/state", "", nil)
	require.Equal(t, 200, a.Code)
	require.Equal(t, 200, b.Code)
	require.Len(t, a.Result().Cookies(), 1, "first REST request must create an anonymous laboratory cookie")
	require.Len(t, b.Result().Cookies(), 1)
	ca, cb := a.Result().Cookies()[0], b.Result().Cookies()[0]
	require.NotEqual(t, ca.Value, cb.Value)
	require.Equal(t, "omenpath_session", ca.Name)
	require.True(t, ca.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, ca.SameSite)
	require.Equal(t, "/", ca.Path)
	require.Equal(t, 2592000, ca.MaxAge)
	for _, cookie := range []*http.Cookie{ca, cb} {
		response := sessionRequest(t, router, "POST", "/api/tutorial/signal", `{"signal":"TUTORIAL_INTRO_COMPLETED"}`, cookie)
		require.Equal(t, 200, response.Code, response.Body.String())
	}
	response := sessionRequest(t, router, "POST", "/api/portals/1/close", `{"confirm":true}`, ca)
	require.Equal(t, 200, response.Code, response.Body.String())
	var sa, sb transport.StateSnapshot
	require.NoError(t, json.Unmarshal(sessionRequest(t, router, "GET", "/api/state", "", ca).Body.Bytes(), &sa))
	require.NoError(t, json.Unmarshal(sessionRequest(t, router, "GET", "/api/state", "", cb).Body.Bytes(), &sb))
	// Tutorial may immediately replace its terminal target; the old instance
	// remains closed only in A, while B still owns its original matching ID 1.
	require.NotNil(t, sa.Slots[0].Portal)
	require.EqualValues(t, 2, sa.Slots[0].Portal.ID)
	require.NotNil(t, sb.Slots[0].Portal)
	require.EqualValues(t, 1, sb.Slots[0].Portal.ID)
	require.Contains(t, sessionRequest(t, router, "GET", "/api/portals/1", "", ca).Body.String(), `"status":"CLOSED"`)
	require.Contains(t, sessionRequest(t, router, "GET", "/api/portals/1", "", cb).Body.String(), `"status":"OPEN"`)
	require.Equal(t, 404, sessionRequest(t, router, "GET", "/api/portals/2", "", cb).Code)
	ea := sessionRequest(t, router, "GET", "/api/events", "", ca)
	eb := sessionRequest(t, router, "GET", "/api/events", "", cb)
	require.Contains(t, ea.Body.String(), "PORTAL_CLOSED")
	require.NotContains(t, eb.Body.String(), "PORTAL_CLOSED")
	require.Empty(t, sessionRequest(t, router, "GET", "/api/state", "", ca).Result().Cookies(), "fresh cookie must be reused without writes/reissue")
}

func TestSessions_DirectWebSocketCreatesCookie(t *testing.T) {
	router := sessionTestRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, response, err := websocket.Dial(ctx, realtimeURL(server.URL), nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	require.Len(t, response.Cookies(), 1, "direct WebSocket handshake must create exactly one cookie")
	require.Equal(t, "omenpath_session", response.Cookies()[0].Name)
	snapshot := receiveRouterSnapshot(t, conn)
	require.Len(t, snapshot.Slots, 7)
	rest := sessionRequest(t, router, "GET", "/api/state", "", response.Cookies()[0])
	require.Equal(t, 200, rest.Code)
	require.Empty(t, rest.Result().Cookies())
}
