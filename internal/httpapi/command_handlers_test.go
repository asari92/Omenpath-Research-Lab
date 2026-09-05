package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/transport"
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

func TestPortalCommandResponse_DerivesDTOAtResolvedSnapshotTimestamp(t *testing.T) {
	now := testutil.BaseTime
	snapshot := httpSnapshot(now)
	snapshot.Simulation.Portals[0].ScheduledCloseAt = now.Add(time.Second)
	manager := &fakeManager{snapshot: snapshot}
	rr := httptest.NewRecorder()
	router, err := NewRouter(manager, config.Default())
	require.NoError(t, err)
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/portals/1/close", strings.NewReader(`{}`)))
	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		GeneratedAt time.Time `json:"generated_at"`
		Slots       []struct {
			Portal *struct {
				TimeRemainingSeconds int64 `json:"time_remaining_seconds"`
			} `json:"portal"`
		} `json:"slots"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, now, body.GeneratedAt)
	require.Equal(t, int64(1), body.Slots[0].Portal.TimeRemainingSeconds)
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
		{"/api/portals/1/close", `null`},
		{"/api/portals/1/close", `[]`},
		{"/api/portals/1/close", `true`},
		{"/api/portals/1/close", `{"confirm":null}`},
		{"/api/portals/1/close", `{"Confirm":true}`},
		{"/api/portals/1/close", `{"confirm":true,"confirm":false}`},
		{"/api/portals/1/close", `{"confirm":"yes"}`},
		{"/api/extraction/open", `{}`},
		{"/api/extraction/open", `{"plane_id":0}`},
		{"/api/extraction/open", `null`},
		{"/api/extraction/open", `{"Plane_ID":1}`},
		{"/api/extraction/open", `{"plane_id":1,"plane_id":2}`},
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

func TestNonDomainRoutingFailuresDoNotReachManager(t *testing.T) {
	for _, tc := range []struct {
		name   string
		path   string
		status int
	}{
		{"unknown route", "/api/not-a-route", http.StatusNotFound},
		{"nonnumeric portal id", "/api/portals/not-a-number/close", http.StatusBadRequest},
		{"zero portal id", "/api/portals/0/close", http.StatusBadRequest},
		{"negative portal id", "/api/portals/-1/close", http.StatusBadRequest},
		{"overflow portal id", "/api/portals/9223372036854775808/close", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
			rr := perform(manager, http.MethodPost, tc.path, `{}`)

			require.Equal(t, tc.status, rr.Code)
			require.Empty(t, manager.commands)
			require.Zero(t, manager.stateCalls)
			require.Zero(t, manager.portalReads)
			require.Zero(t, manager.eventsReads)
		})
	}
}

func TestInternalFailure_ReturnsOpaque500(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: errors.New("database password secret")}
	rr := perform(manager, http.MethodPost, "/api/portals/1/stabilize", `{}`)
	require.Equal(t, 500, rr.Code)
	require.NotContains(t, rr.Body.String(), "database password secret")
}

func TestCompositeInfrastructureErrorsAreOpaque500(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"domain conflict plus restore failure", errors.Join(domain.ErrConfirmationRequired, errors.New("rng restore secret"))},
		{"not found plus persistence failure", errors.Join(engine.ErrPortalNotFound, errors.New("database secret"))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: tc.err}
			rr := perform(manager, http.MethodPost, "/api/portals/1/close", `{}`)
			require.Equal(t, http.StatusInternalServerError, rr.Code)
			require.JSONEq(t, `{"error":{"code":"INTERNAL_ERROR","message":"internal server error","confirmable":false}}`, rr.Body.String())
			require.NotContains(t, rr.Body.String(), "secret")
			require.NotContains(t, rr.Body.String(), "confirmation required")
		})
	}
}

func TestCleanWrappedDomainErrorsPreserveMapping(t *testing.T) {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{
			"confirmation", fmt.Errorf("command context: %w", domain.ErrConfirmationRequired), http.StatusConflict,
			`{"error":{"code":"CONFIRMATION_REQUIRED","message":"confirmation required","confirmable":true}}`,
		},
		{
			"portal not found", fmt.Errorf("command context: %w", engine.ErrPortalNotFound), http.StatusNotFound,
			`{"error":{"code":"PORTAL_NOT_FOUND","message":"portal not found","confirmable":false}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: tc.err}
			rr := perform(manager, http.MethodPost, "/api/portals/1/close", `{}`)
			require.Equal(t, tc.status, rr.Code)
			require.JSONEq(t, tc.body, rr.Body.String())
			require.NotContains(t, rr.Body.String(), "command context")
		})
	}
}

func TestSingleCauseJoinedDomainErrorPreservesMapping(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{"direct single join", errors.Join(domain.ErrConfirmationRequired)},
		{"wrapped single join", fmt.Errorf("command context: %w", errors.Join(domain.ErrConfirmationRequired))},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: tc.err}
			rr := perform(manager, http.MethodPost, "/api/portals/1/close", `{}`)
			require.Equal(t, http.StatusConflict, rr.Code)
			require.JSONEq(t, `{"error":{"code":"CONFIRMATION_REQUIRED","message":"confirmation required","confirmable":true}}`, rr.Body.String())
			require.NotContains(t, rr.Body.String(), "command context")
		})
	}
}

func TestDomainConflictMapping_UsesSharedTransportDescriptor(t *testing.T) {
	descriptor, ok := transport.DescribeDomainError(domain.ErrPortalCriticalRisk)
	require.True(t, ok)
	require.Equal(t, "PORTAL_CRITICAL_RISK", descriptor.Code)
	require.Equal(t, "portal risk is critical", descriptor.Message)
	require.False(t, descriptor.Confirmable)
}
