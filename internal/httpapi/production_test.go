package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductionHandler_HealthBypassesSession(t *testing.T) {
	webRoot := productionWebRoot(t)
	apiCalled := false
	api := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		apiCalled = true
	})

	handler, err := NewProductionHandler(api, webRoot)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	require.JSONEq(t, `{"status":"ok"}`, recorder.Body.String())
	require.False(t, apiCalled)
}

func TestProductionHandler_ServesAssetsAndSPAFallback(t *testing.T) {
	handler, err := NewProductionHandler(http.NotFoundHandler(), productionWebRoot(t))
	require.NoError(t, err)

	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	require.Equal(t, http.StatusOK, asset.Code)
	require.Equal(t, "console.log('omenpath')", asset.Body.String())

	spa := httptest.NewRecorder()
	handler.ServeHTTP(spa, httptest.NewRequest(http.MethodGet, "/portals/42", nil))
	require.Equal(t, http.StatusOK, spa.Code)
	require.Contains(t, spa.Body.String(), `<div id="root"></div>`)
}

func TestProductionHandler_DelegatesAPIAndWebSocket(t *testing.T) {
	seen := make([]string, 0, 2)
	api := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	handler, err := NewProductionHandler(api, productionWebRoot(t))
	require.NoError(t, err)

	for _, path := range []string{"/api/state", "/ws/lab"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, http.StatusNoContent, recorder.Code)
	}
	require.Equal(t, []string{"/api/state", "/ws/lab"}, seen)
}

func TestProductionHandler_RejectsMissingStaticAsset(t *testing.T) {
	handler, err := NewProductionHandler(http.NotFoundHandler(), productionWebRoot(t))
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/assets/missing.js", nil))
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func productionWebRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "assets"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "index.html"), []byte(`<div id="root"></div>`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "assets", "app.js"), []byte("console.log('omenpath')"), 0o644))
	return root
}
