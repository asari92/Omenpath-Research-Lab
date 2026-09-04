package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/testutil"
)

func (m *fakeManager) StartTutorial(context.Context) error {
	return m.runCommand("START_TUTORIAL", 0, false)
}

func (m *fakeManager) ResetTutorial(context.Context) error {
	return m.runCommand("RESET_TUTORIAL", 0, false)
}

func (m *fakeManager) TutorialSignal(_ context.Context, signal domain.TutorialSignal, portalID *int64) error {
	id := int64(0)
	if portalID != nil {
		id = *portalID
	}
	return m.runCommand("TUTORIAL_SIGNAL:"+string(signal), id, false)
}

func (m *fakeManager) StartLive(context.Context) error {
	return m.runCommand("START_LIVE", 0, false)
}

func TestTutorialSignal_StrictShapeAndClosedEnum(t *testing.T) {
	tests := []struct {
		body string
	}{
		{`{}`},
		{`{"signal":"UNKNOWN"}`},
		{`{"signal":"PORTAL_DETAILS_OPENED"}`},
		{`{"signal":"TUTORIAL_INTRO_COMPLETED","portal_id":1}`},
		{`{"signal":"EVENT_LOG_OPENED","portal_id":1}`},
		{`{"signal":"PORTAL_DETAILS_OPENED","portal_id":0}`},
		{`{"signal":"PORTAL_DETAILS_OPENED","portal_id":1,"extra":true}`},
	}
	for _, tc := range tests {
		manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
		rr := perform(manager, http.MethodPost, "/api/tutorial/signal", tc.body)
		require.Equal(t, http.StatusBadRequest, rr.Code, tc.body)
		require.Empty(t, manager.commands, tc.body)
	}
}

func TestTutorialSignal_UnexpectedValidSignalCreatesActionRejected(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), commandErr: engine.ErrInvalidTutorialSignal}
	rr := perform(manager, http.MethodPost, "/api/tutorial/signal", `{"signal":"EVENT_LOG_OPENED"}`)
	require.Equal(t, http.StatusConflict, rr.Code)
	require.Len(t, manager.commands, 1)
}

func TestTutorialAPI_GETsNeverMutateProgress(t *testing.T) {
	manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	for _, path := range []string{"/api/state", "/api/events", "/api/portals/1"} {
		rr := perform(manager, http.MethodGet, path, "")
		require.Equal(t, http.StatusOK, rr.Code)
	}
	require.Empty(t, manager.commands)
}

func TestTutorialLifecycleRoutes_ReturnFreshAuthoritativeState(t *testing.T) {
	for _, path := range []string{"/api/tutorial/start", "/api/tutorial/reset", "/api/live/start"} {
		manager := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
		rr := perform(manager, http.MethodPost, path, `{}`)
		require.Equal(t, http.StatusOK, rr.Code, path)
		require.Len(t, manager.commands, 1, path)
	}
}
