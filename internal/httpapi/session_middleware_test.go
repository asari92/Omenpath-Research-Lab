package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/labruntime"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/session"
	"omenpath-lab/testutil"
)

type resolveFunc func(context.Context, string) (session.Resolution, error)

func (f resolveFunc) Resolve(c context.Context, s string) (session.Resolution, error) { return f(c, s) }
func middlewareFixture(t *testing.T) (*session.Service, *labruntime.Registry, *testutil.FakeClock) {
	t.Helper()
	ctx := context.Background()
	cfg := config.Default()
	c := testutil.NewFakeClock(testutil.BaseTime)
	store, err := persistence.Open(ctx, t.TempDir()+"/session.sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	service, err := session.New(store, cfg, c, nil)
	require.NoError(t, err)
	registry, err := labruntime.New(store, cfg, c)
	require.NoError(t, err)
	t.Cleanup(registry.Close)
	return service, registry, c
}
func TestSessionMiddleware_CookieRefreshReplacementAndTypedScope(t *testing.T) {
	service, registry, c := middlewareFixture(t)
	cfg := config.Default()
	var id persistence.LabID
	var runtime *labruntime.Runtime
	handler := sessionMiddleware(service, registry, cfg, true)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope, ok := r.Context().Value(scopeKey{}).(*requestScope)
		require.True(t, ok)
		require.NotNil(t, scope.lease)
		require.Same(t, scope.runtime.Manager, scope.manager)
		id = scope.labID
		runtime = scope.runtime
		ok, err := registry.DeleteIfIdle(id, func() error { t.Error("REST lease not held"); return nil })
		require.NoError(t, err)
		require.False(t, ok)
		w.WriteHeader(204)
	}))
	response := sessionRequest(t, handler, "GET", "/api/state", "", nil)
	require.Equal(t, 204, response.Code)
	require.Len(t, response.Result().Cookies(), 1)
	cookie := response.Result().Cookies()[0]
	firstID, firstRuntime := id, runtime
	require.True(t, cookie.Secure)
	require.True(t, cookie.HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookie.SameSite)
	require.Equal(t, 2592000, cookie.MaxAge)
	require.Equal(t, c.Now().Add(30*24*time.Hour), cookie.Expires)
	response = sessionRequest(t, handler, "GET", "/api/state", "", cookie)
	require.Empty(t, response.Result().Cookies())
	require.Equal(t, firstID, id)
	require.Same(t, firstRuntime, runtime)
	c.Advance(12 * time.Hour)
	response = sessionRequest(t, handler, "GET", "/api/state", "", cookie)
	require.Len(t, response.Result().Cookies(), 1)
	require.Equal(t, cookie.Value, response.Result().Cookies()[0].Value)
	require.Equal(t, c.Now().Add(cfg.SessionTTL), response.Result().Cookies()[0].Expires)
	c.Advance(31 * 24 * time.Hour)
	response = sessionRequest(t, handler, "GET", "/api/state", "", cookie)
	require.Len(t, response.Result().Cookies(), 1)
	require.NotEqual(t, cookie.Value, response.Result().Cookies()[0].Value)
	require.NotEqual(t, firstID, id)
	invalid := &http.Cookie{Name: cfg.SessionCookieName, Value: "invalid-token"}
	response = sessionRequest(t, handler, "GET", "/api/state", "", invalid)
	require.Equal(t, 204, response.Code)
	require.Len(t, response.Result().Cookies(), 1)
}
func TestSessionMiddleware_FailuresOpaqueAndPublicRoutesUntouched(t *testing.T) {
	_, registry, _ := middlewareFixture(t)
	calls := 0
	resolver := resolveFunc(func(context.Context, string) (session.Resolution, error) {
		calls++
		return session.Resolution{}, errors.New("secret-token sqlite-path")
	})
	router, err := NewRouter(resolver, registry, config.Default(), false)
	require.NoError(t, err)
	for _, path := range []string{"/help", "/assets/a.png", "/ai-worklog"} {
		response := sessionRequest(t, router, "GET", path, "", nil)
		require.Equal(t, 404, response.Code)
		require.Empty(t, response.Result().Cookies())
	}
	require.Zero(t, calls)
	for _, path := range []string{"/api/state", "/api/unknown", "/ws/lab"} {
		response := sessionRequest(t, router, "GET", path, "", nil)
		require.Equal(t, 500, response.Code)
		require.NotContains(t, response.Body.String(), "secret-token")
		require.NotContains(t, response.Body.String(), "sqlite-path")
		require.Empty(t, response.Result().Cookies())
	}
	require.Equal(t, 3, calls)
	registry.Close()
	good := resolveFunc(func(context.Context, string) (session.Resolution, error) {
		return session.Resolution{LabID: "00000000000000000000000000000001"}, nil
	})
	handler := sessionMiddleware(good, registry, config.Default(), false)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("closed runtime reached handler") }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest("GET", "/api/state", nil))
	require.Equal(t, 500, response.Code)
	require.NotContains(t, response.Body.String(), "registry")
}

func TestNewRouter_ValidatesSessionDependencies(t *testing.T) {
	service, registry, _ := middlewareFixture(t)
	var nilService *session.Service
	for _, resolver := range []SessionResolver{nil, nilService} {
		router, err := NewRouter(resolver, registry, config.Default(), false)
		require.Error(t, err)
		require.Nil(t, router)
	}
	router, err := NewRouter(service, nil, config.Default(), false)
	require.Error(t, err)
	require.Nil(t, router)
	cfg := config.Default()
	cfg.MaxActivePortals = 0
	router, err = NewRouter(service, registry, cfg, false)
	require.Error(t, err)
	require.Nil(t, router)
}
