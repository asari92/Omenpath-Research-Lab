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
	snapshot    persistence.Snapshot
	events      []domain.Event
	stateErr    error
	portalErr   error
	eventsErr   error
	stateCalls  int
	portalReads int
	eventsReads int
	commandErr  error
	commandHook func(action string, id int64, confirm bool) error
	commands    []commandCall
	updates     chan struct{}
}

var inertManagerUpdates = make(chan struct{})

type commandCall struct {
	action  string
	id      int64
	confirm bool
}

func (m *fakeManager) State(context.Context) (persistence.Snapshot, error) {
	m.stateCalls++
	return m.snapshot, m.stateErr
}
func (m *fakeManager) Updates() <-chan struct{} {
	if m.updates != nil {
		return m.updates
	}
	return inertManagerUpdates
}
func (m *fakeManager) Portal(_ context.Context, id int64) (domain.Portal, []domain.Event, error) {
	m.portalReads++
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

func (m *fakeManager) PortalState(_ context.Context, id int64) (persistence.Snapshot, []domain.Event, error) {
	m.portalReads++
	if m.portalErr != nil {
		return persistence.Snapshot{}, nil, m.portalErr
	}
	for _, portal := range m.snapshot.Simulation.Portals {
		if portal.ID == id {
			return m.snapshot, domain.PortalHistory(m.events, id), nil
		}
	}
	return persistence.Snapshot{}, nil, engine.ErrPortalNotFound
}
func (m *fakeManager) Events(context.Context) ([]domain.Event, error) {
	m.eventsReads++
	return m.events, m.eventsErr
}
func (m *fakeManager) runCommand(action string, id int64, confirm bool) error {
	m.commands = append(m.commands, commandCall{action: action, id: id, confirm: confirm})
	if m.commandHook != nil {
		return m.commandHook(action, id, confirm)
	}
	return m.commandErr
}
func (m *fakeManager) Stabilize(_ context.Context, id int64) error {
	return m.runCommand("STABILIZE", id, false)
}
func (m *fakeManager) ClosePortal(_ context.Context, id int64, confirm bool) error {
	return m.runCommand("CLOSE", id, confirm)
}
func (m *fakeManager) SendObserver(_ context.Context, id int64, confirm bool) error {
	return m.runCommand("SEND", id, confirm)
}
func (m *fakeManager) RecallObserver(_ context.Context, id int64, confirm bool) error {
	return m.runCommand("RECALL", id, confirm)
}
func (m *fakeManager) OpenExtraction(_ context.Context, id int64) error {
	return m.runCommand("OPEN_EXTRACTION", id, false)
}

func httpSnapshot(now time.Time) persistence.Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: "Plane", Aliases: []string{}, CatalogTier: "core"}
	}
	p := testutil.NewPortalBuilder().Build()
	p.OpenedAt, p.CreatedAt, p.UpdatedAt, p.EnergyBaseAt = now, now, now, now
	p.ScheduledCloseAt = now.Add(time.Minute)
	lastTick := now
	return persistence.Snapshot{Simulation: domain.SimulationState{
		Lab: domain.LabState{EnergyBase: 100, EnergyBaseAt: now}, Portals: []domain.Portal{p},
		Planes: planes, Observers: domain.NewObserverRoster(10, now), NextPortalID: 2,
		NaturalSpawn: domain.NaturalSpawnState{Paused: true}, LastTickAt: &lastTick,
	}, App: domain.AppState{Mode: domain.ModeTutorial}}
}

func newReadRouter(manager *fakeManager, _ time.Time) http.Handler {
	router, err := NewRouter(manager, config.Default())
	if err != nil {
		panic(err)
	}
	return router
}

func TestNewRouter_RejectsNilManagerAndInvalidConfigAtConstruction(t *testing.T) {
	valid := config.Default()
	var typedNil *fakeManager
	invalidSlots := valid
	invalidSlots.MaxActivePortals = 0
	invalidTransit := valid
	invalidTransit.CreatureTransit = 0
	for _, tc := range []struct {
		name    string
		manager Manager
		cfg     config.Config
	}{
		{"nil manager", nil, valid},
		{"typed nil manager", typedNil, valid},
		{"invalid slots", &fakeManager{}, invalidSlots},
		{"invalid creature transit", &fakeManager{}, invalidTransit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				router, err := NewRouter(tc.manager, tc.cfg)
				require.Error(t, err)
				require.Nil(t, router)
			})
		})
	}
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

func TestGetState_DerivesDTOAtSnapshotCatchUpTimestamp(t *testing.T) {
	now := testutil.BaseTime
	snapshot := httpSnapshot(now)
	snapshot.Simulation.Portals[0].ScheduledCloseAt = now.Add(time.Second)
	manager := &fakeManager{snapshot: snapshot}
	rr := httptest.NewRecorder()
	router, err := NewRouter(manager, config.Default())
	require.NoError(t, err)
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		GeneratedAt time.Time `json:"generated_at"`
		Slots       []struct {
			Portal *struct {
				Status               domain.PortalStatus `json:"status"`
				TimeRemainingSeconds int64               `json:"time_remaining_seconds"`
			} `json:"portal"`
		} `json:"slots"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, now, body.GeneratedAt)
	require.Equal(t, domain.PortalStatusOpen, body.Slots[0].Portal.Status)
	require.Equal(t, int64(1), body.Slots[0].Portal.TimeRemainingSeconds)
}

type coherenceManager struct {
	atomicSnapshot persistence.Snapshot
	atomicHistory  []domain.Event
	staleSnapshot  persistence.Snapshot
	portalCalls    int
	stateCalls     int
	atomicCalls    int
}

func (m *coherenceManager) State(context.Context) (persistence.Snapshot, error) {
	m.stateCalls++
	return m.staleSnapshot, nil
}
func (m *coherenceManager) Updates() <-chan struct{} { return inertManagerUpdates }
func (m *coherenceManager) Portal(context.Context, int64) (domain.Portal, []domain.Event, error) {
	m.portalCalls++
	return m.atomicSnapshot.Simulation.Portals[0], m.atomicHistory[:1], nil
}
func (m *coherenceManager) PortalState(context.Context, int64) (persistence.Snapshot, []domain.Event, error) {
	m.atomicCalls++
	return m.atomicSnapshot, m.atomicHistory, nil
}
func (m *coherenceManager) Events(context.Context) ([]domain.Event, error)    { return nil, nil }
func (m *coherenceManager) Stabilize(context.Context, int64) error            { return nil }
func (m *coherenceManager) ClosePortal(context.Context, int64, bool) error    { return nil }
func (m *coherenceManager) SendObserver(context.Context, int64, bool) error   { return nil }
func (m *coherenceManager) RecallObserver(context.Context, int64, bool) error { return nil }
func (m *coherenceManager) OpenExtraction(context.Context, int64) error       { return nil }
func (m *coherenceManager) StartTutorial(context.Context) error               { return nil }
func (m *coherenceManager) ResetTutorial(context.Context) error               { return nil }
func (m *coherenceManager) TutorialSignal(context.Context, domain.TutorialSignal, *int64) error {
	return nil
}
func (m *coherenceManager) StartLive(context.Context) error { return nil }

func TestGetPortal_UsesOneResolvedSnapshotHistoryBoundary(t *testing.T) {
	now := testutil.BaseTime
	open := httpSnapshot(now)
	portalID := int64(1)
	opened := domain.Event{ID: 1, EventType: domain.EventPortalOpened, PortalID: &portalID, Message: "opened", PayloadJSON: `{}`, CreatedAt: now}
	closed := httpSnapshot(now.Add(time.Second))
	closedAt := now.Add(time.Second)
	closed.Simulation.Portals[0].Status = domain.PortalStatusClosed
	closed.Simulation.Portals[0].TerminationReason = domain.TerminationManualClose
	closed.Simulation.Portals[0].ClosedAt = &closedAt
	closed.Simulation.Portals[0].UpdatedAt = closedAt
	manager := &coherenceManager{
		atomicSnapshot: open,
		atomicHistory:  []domain.Event{opened},
		staleSnapshot:  closed,
	}
	rr := httptest.NewRecorder()
	router, err := NewRouter(manager, config.Default())
	require.NoError(t, err)
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/portals/1", nil))
	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		GeneratedAt time.Time `json:"generated_at"`
		Portal      struct {
			Status domain.PortalStatus `json:"status"`
		} `json:"portal"`
		History []struct {
			EventType domain.EventType `json:"event_type"`
		} `json:"history"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	require.Equal(t, now, body.GeneratedAt)
	require.Equal(t, domain.PortalStatusOpen, body.Portal.Status)
	require.Equal(t, []domain.EventType{domain.EventPortalOpened}, []domain.EventType{body.History[0].EventType})
	require.Equal(t, 1, manager.atomicCalls)
	require.Zero(t, manager.portalCalls)
	require.Zero(t, manager.stateCalls)
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
	require.Equal(t, 1, m.stateCalls)
	require.Equal(t, 1, m.portalReads)
	require.Equal(t, 0, m.snapshot.App.TutorialStep)
}

func TestReadEndpoint_InternalFailureIsOpaque(t *testing.T) {
	m := &fakeManager{snapshot: httpSnapshot(testutil.BaseTime), stateErr: errors.New("sqlite secret")}
	rr := httptest.NewRecorder()
	newReadRouter(m, testutil.BaseTime).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	require.Equal(t, 500, rr.Code)
	require.NotContains(t, rr.Body.String(), "sqlite secret")
}
