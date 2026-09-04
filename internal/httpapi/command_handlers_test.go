package httpapi

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/testutil"
)

func perform(manager *fakeManager, method, path, body string) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	newReadRouter(manager, testutil.BaseTime).ServeHTTP(rr, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rr
}

func TestPortalCommandRoutes_ReturnFreshState(t *testing.T) {
	for _, tc := range []struct{ path, action string }{
		{"/api/portals/1/stabilize", "STABILIZE"},
		{"/api/portals/1/close", "CLOSE"},
		{"/api/portals/1/send-observer", "SEND"},
		{"/api/portals/1/recall-observer", "RECALL"},
	} {
		t.Run(tc.action, func(t *testing.T) {
			manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
			rr := perform(manager, http.MethodPost, tc.path, `{}`)
			require.Equal(t, 200, rr.Code)
			require.Len(t, manager.commands, 1)
			require.Equal(t, tc.action, manager.commands[0].action)
			require.Equal(t, int64(1), manager.commands[0].id)
			require.Equal(t, 1, manager.stateCalls)
			require.Contains(t, rr.Body.String(), `"generated_at"`)
		})
	}
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	rr := perform(manager, http.MethodPost, "/api/extraction/open", `{"plane_id":1}`)
	require.Equal(t, 200, rr.Code)
	require.Equal(t, []commandCall{{action: "OPEN_EXTRACTION", id: 1}}, manager.commands)
	require.Equal(t, 1, manager.stateCalls)
}

func TestCloseConfirmationFlow_Returns409ThenSucceeds(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	manager.commandHook = func(action string, _ int64, confirm bool) error {
		if action == "CLOSE" && !confirm {
			return domain.ErrConfirmationRequired
		}
		return nil
	}
	first := perform(manager, http.MethodPost, "/api/portals/1/close", `{}`)
	require.Equal(t, 409, first.Code)
	require.JSONEq(t, `{"error":{"code":"CONFIRMATION_REQUIRED","message":"confirmation required","confirmable":true}}`, first.Body.String())
	second := perform(manager, http.MethodPost, "/api/portals/1/close", `{"confirm":true}`)
	require.Equal(t, 200, second.Code)
	require.Equal(t, []bool{false, true}, []bool{manager.commands[0].confirm, manager.commands[1].confirm})
}

func TestDomainConflictMapping(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{domain.ErrPortalCriticalRisk, "PORTAL_CRITICAL_RISK"},
		{domain.ErrInsufficientLabEnergy, "INSUFFICIENT_LAB_ENERGY"},
		{domain.ErrPortalDirectionConflict, "PORTAL_DIRECTION_CONFLICT"},
		{domain.ErrPortalCreaturesPresent, "PORTAL_CREATURES_PRESENT"},
	} {
		manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: tc.err}
		rr := perform(manager, http.MethodPost, "/api/portals/1/send-observer", `{}`)
		require.Equal(t, 409, rr.Code)
		require.Contains(t, rr.Body.String(), `"code":"`+tc.code+`"`)
	}
}

func TestCommandRoute_UnknownPortalIs404(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: engine.ErrPortalNotFound}
	rr := perform(manager, http.MethodPost, "/api/portals/999/close", `{}`)
	require.Equal(t, 404, rr.Code)
	require.Len(t, manager.commands, 1)
	require.Equal(t, int64(999), manager.commands[0].id)
}

func TestExtractionRoute_UnknownPlaneIs404(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: engine.ErrPlaneNotFound}
	rr := perform(manager, http.MethodPost, "/api/extraction/open", `{"plane_id":999}`)
	require.Equal(t, 404, rr.Code)
	require.Len(t, manager.commands, 1)
}

func TestCommandRoute_StrictJSONRejectsUnknownAndTrailingValues(t *testing.T) {
	for _, tc := range []struct{ path, body string }{
		{"/api/portals/1/close", `{"unknown":true}`},
		{"/api/portals/1/close", `{} {}`},
		{"/api/portals/1/close", ``},
		{"/api/portals/1/close", `{"confirm":"yes"}`},
		{"/api/extraction/open", `{}`},
		{"/api/extraction/open", `{"plane_id":0}`},
		{"/api/extraction/open", `{"plane_id":1,"extra":2}`},
	} {
		manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
		rr := perform(manager, http.MethodPost, tc.path, tc.body)
		require.Equal(t, 400, rr.Code, "%s %s", tc.path, tc.body)
		require.Empty(t, manager.commands)
	}
}

func TestMalformedRequest_DoesNotCreateActionRejected(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	rr := perform(manager, http.MethodPost, "/api/portals/1/send-observer", `{bad`)
	require.Equal(t, 400, rr.Code)
	require.Empty(t, manager.commands)
}

func TestWrongMethod_Returns405(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/portals/1/close"},
		{http.MethodPost, "/api/state"},
	} {
		rr := perform(manager, tc.method, tc.path, `{}`)
		require.Equal(t, 405, rr.Code)
	}
	require.Empty(t, manager.commands)
}

func TestInternalFailure_ReturnsOpaque500(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: errors.New("database password secret")}
	rr := perform(manager, http.MethodPost, "/api/portals/1/stabilize", `{}`)
	require.Equal(t, 500, rr.Code)
	require.NotContains(t, rr.Body.String(), "database password secret")
}
