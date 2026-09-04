package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

type fakeManager struct {
	snapshot   persistence.Snapshot
	events     []domain.Event
	stateErr   error
	portalErr  error
	eventsErr  error
	stateCalls int
}

func (m *fakeManager) State(context.Context) (persistence.Snapshot, error) {
	m.stateCalls++
	return m.snapshot, m.stateErr
}
func (m *fakeManager) Portal(_ context.Context, id int64) (domain.Portal, []domain.Event, error) {
	if m.portalErr != nil {
		return domain.Portal{}, nil, m.portalErr
	}
	for _, portal := range m.snapshot.Simulation.Portals {
		if portal.ID == id {
			return portal, domain.PortalHistory(m.events, id), nil
		}
	}
	return domain.Portal{}, nil, engine.ErrPortalNotFound
}
func (m *fakeManager) Events(context.Context) ([]domain.Event, error)    { return m.events, m.eventsErr }
func (m *fakeManager) Stabilize(context.Context, int64) error            { return nil }
func (m *fakeManager) ClosePortal(context.Context, int64, bool) error    { return nil }
func (m *fakeManager) SendObserver(context.Context, int64, bool) error   { return nil }
func (m *fakeManager) RecallObserver(context.Context, int64, bool) error { return nil }
func (m *fakeManager) OpenExtraction(context.Context, int64) error       { return nil }

func httpSnapshot(now time.Time) persistence.Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: "Plane", Aliases: []string{}, CatalogTier: "core"}
	}
	p := testutil.NewPortalBuilder().Build()
	return persistence.Snapshot{Simulation: domain.SimulationState{
		Lab: domain.LabState{EnergyBase: 100, EnergyBaseAt: now}, Portals: []domain.Portal{p},
		Planes: planes, Observers: domain.NewObserverRoster(10, now), NextPortalID: 2,
		NaturalSpawn: domain.NaturalSpawnState{Paused: true},
	}, App: domain.AppState{Mode: domain.ModeTutorial}}
}

func newReadRouter(manager *fakeManager, now time.Time) http.Handler {
	return NewRouter(manager, config.Default(), testutil.NewFakeClock(now))
}

func TestGetState_ReturnsAuthoritativeSnapshot(t *testing.T) {
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	rr := httptest.NewRecorder()
	newReadRouter(m, testutil.BaseTime).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Len(t, body["slots"], 7)
}

func TestGetPortal_ReturnsDetailsOr404(t *testing.T) {
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	router := newReadRouter(m, testutil.BaseTime)
	for _, tc := range []struct {
		path   string
		status int
	}{{"/api/portals/1", 200}, {"/api/portals/999", 404}} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))
		require.Equal(t, tc.status, rr.Code)
	}
}

func TestGetEvents_ReturnsChronologicalGlobalLog(t *testing.T) {
	later := testutil.BaseTime.Add(time.Second)
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), events: []domain.Event{
		{ID: 2, EventType: domain.EventPortalClosed, Message: "later", PayloadJSON: `{}`, CreatedAt: later},
		{ID: 1, EventType: domain.EventPortalOpened, Message: "first", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime},
	}}
	rr := httptest.NewRecorder()
	newReadRouter(m, testutil.BaseTime).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/events", nil))
	require.Equal(t, 200, rr.Code)
	var body []struct {
		ID int64 `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, []int64{1, 2}, []int64{body[0].ID, body[1].ID})
}

func TestReadEndpoints_DoNotAdvanceTutorial(t *testing.T) {
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime)}
	router := newReadRouter(m, testutil.BaseTime)
	for _, path := range []string{"/api/state", "/api/portals/1", "/api/events"} {
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		require.Equal(t, 200, rr.Code)
	}
	require.Equal(t, 2, m.stateCalls)
	require.Equal(t, 0, m.snapshot.App.TutorialStep)
}

func TestReadEndpoint_InternalFailureIsOpaque(t *testing.T) {
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), stateErr: errors.New("sqlite secret")}
	rr := httptest.NewRecorder()
	newReadRouter(m, testutil.BaseTime).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	require.Equal(t, 500, rr.Code)
	require.NotContains(t, rr.Body.String(), "sqlite secret")
}
