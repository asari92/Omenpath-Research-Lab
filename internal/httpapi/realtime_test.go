package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/transport"
	"omenpath-lab/testutil"
)

type realtimeManager struct {
	mu       sync.Mutex
	snapshot persistence.Snapshot
	events   []domain.Event
	updates  chan struct{}
	reject   error
	states   int
}

func newRealtimeManager() *realtimeManager {
	return &realtimeManager{snapshot: httpSnapshot(testutil.BaseTime), updates: make(chan struct{}, 1)}
}

func (m *realtimeManager) State(context.Context) (persistence.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states++
	return m.snapshot, nil
}

func (m *realtimeManager) Updates() <-chan struct{} { return m.updates }

func (m *realtimeManager) Portal(context.Context, int64) (domain.Portal, []domain.Event, error) {
	return domain.Portal{}, nil, engine.ErrPortalNotFound
}

func (m *realtimeManager) PortalState(context.Context, int64) (persistence.Snapshot, []domain.Event, error) {
	return persistence.Snapshot{}, nil, engine.ErrPortalNotFound
}

func (m *realtimeManager) Events(context.Context) ([]domain.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]domain.Event(nil), m.events...), nil
}

func (m *realtimeManager) command() error {
	m.mu.Lock()
	err := m.reject
	if err == nil {
		now := m.snapshot.Simulation.LastTickAt.Add(time.Second)
		m.snapshot.Simulation.Lab.EnergyBase--
		m.snapshot.Simulation.Lab.EnergyBaseAt = now
		m.snapshot.Simulation.LastTickAt = &now
	} else {
		m.events = append(m.events, domain.Event{ID: int64(len(m.events) + 1), EventType: domain.EventActionRejected, Message: "rejected", PayloadJSON: `{}`, CreatedAt: *m.snapshot.Simulation.LastTickAt})
	}
	m.mu.Unlock()
	select {
	case m.updates <- struct{}{}:
	default:
	}
	return err
}

func (m *realtimeManager) Stabilize(context.Context, int64) error         { return m.command() }
func (m *realtimeManager) ClosePortal(context.Context, int64, bool) error { return m.command() }
func (m *realtimeManager) SendObserver(context.Context, int64, bool) error {
	return m.command()
}
func (m *realtimeManager) RecallObserver(context.Context, int64, bool) error {
	return m.command()
}
func (m *realtimeManager) OpenExtraction(context.Context, int64) error { return m.command() }
func (m *realtimeManager) StartTutorial(context.Context) error         { return m.command() }
func (m *realtimeManager) ResetTutorial(context.Context) error         { return m.command() }
func (m *realtimeManager) TutorialSignal(context.Context, domain.TutorialSignal, *int64) error {
	return m.command()
}
func (m *realtimeManager) StartLive(context.Context) error { return m.command() }

func (m *realtimeManager) stateCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.states
}

func realtimeURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + "/ws/lab"
}

func connectRouterWebSocket(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, realtimeURL(serverURL), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test complete") })
	return conn
}

func receiveRouterSnapshot(t *testing.T, conn *websocket.Conn) transport.StateSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var snapshot transport.StateSnapshot
	require.NoError(t, wsjson.Read(ctx, conn, &snapshot))
	return snapshot
}

func TestWebSocket_SuccessfulActionBroadcastsImmediately(t *testing.T) {
	manager := newRealtimeManager()
	router, err := newManagerRouter(manager, config.Default())
	require.NoError(t, err)
	t.Cleanup(router.Close)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	conn := connectRouterWebSocket(t, server.URL)
	initial := receiveRouterSnapshot(t, conn)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/portals/1/stabilize", strings.NewReader(`{}`))
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusOK, response.StatusCode)

	updated := receiveRouterSnapshot(t, conn)
	require.Equal(t, initial.GeneratedAt.Add(time.Second), updated.GeneratedAt)
	require.Equal(t, initial.Lab.CurrentEnergy-1, updated.Lab.CurrentEnergy)
}

func TestWebSocket_RejectedDomainActionBroadcastsPersistedEvent(t *testing.T) {
	manager := newRealtimeManager()
	manager.reject = domain.ErrPortalCriticalRisk
	router, err := newManagerRouter(manager, config.Default())
	require.NoError(t, err)
	t.Cleanup(router.Close)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	conn := connectRouterWebSocket(t, server.URL)
	initial := receiveRouterSnapshot(t, conn)

	request, err := http.NewRequest(http.MethodPost, server.URL+"/api/portals/1/send-observer", strings.NewReader(`{}`))
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, http.StatusConflict, response.StatusCode)

	updated := receiveRouterSnapshot(t, conn)
	require.Equal(t, initial, updated)
	events, err := manager.Events(context.Background())
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, domain.EventActionRejected, events[0].EventType)
}

func TestRouterClose_OwnsWebSocketLifecycle(t *testing.T) {
	manager := newRealtimeManager()
	router, err := newManagerRouter(manager, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	conn := connectRouterWebSocket(t, server.URL)
	_ = receiveRouterSnapshot(t, conn)

	closer, ok := any(router).(interface{ Close() })
	require.True(t, ok, "router must expose lifecycle ownership")
	if !ok {
		return
	}
	require.NotPanics(t, func() {
		closer.Close()
		closer.Close()
	})

	readCtx, cancelRead := context.WithTimeout(context.Background(), time.Second)
	defer cancelRead()
	var snapshot transport.StateSnapshot
	require.Error(t, wsjson.Read(readCtx, conn, &snapshot), "Close must promptly disconnect an active client")

	readsAfterClose := manager.stateCalls()
	manager.updates <- struct{}{}
	require.Never(t, func() bool { return manager.stateCalls() != readsAfterClose }, 100*time.Millisecond, 10*time.Millisecond)

	dialCtx, cancelDial := context.WithTimeout(context.Background(), time.Second)
	defer cancelDial()
	reconnected, response, err := websocket.Dial(dialCtx, realtimeURL(server.URL), nil)
	if reconnected != nil {
		_ = reconnected.CloseNow()
	}
	require.Error(t, err)
	require.NotNil(t, response)
	require.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
}
