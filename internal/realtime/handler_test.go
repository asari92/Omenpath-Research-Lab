package realtime

import (
	"context"
	"encoding/json"
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
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/transport"
	"omenpath-lab/testutil"
)

type fakeStateSource struct {
	mu       sync.Mutex
	snapshot persistence.Snapshot
	updates  chan struct{}
}

func newFakeStateSource(snapshot persistence.Snapshot) *fakeStateSource {
	return &fakeStateSource{snapshot: snapshot, updates: make(chan struct{}, 1)}
}

func (s *fakeStateSource) State(context.Context) (persistence.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot, nil
}

func (s *fakeStateSource) Updates() <-chan struct{} { return s.updates }

func (s *fakeStateSource) replace(snapshot persistence.Snapshot) {
	s.mu.Lock()
	s.snapshot = snapshot
	s.mu.Unlock()
	select {
	case s.updates <- struct{}{}:
	default:
	}
}

func realtimeSnapshot(at time.Time) persistence.Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{
			ID: int64(i + 1), Name: "Plane", Aliases: []string{}, CatalogTier: "core",
		}
	}
	portal := testutil.NewPortalBuilder().Build()
	portal.CreatedAt = at
	portal.OpenedAt = at
	portal.UpdatedAt = at
	portal.EnergyBaseAt = at
	portal.ScheduledCloseAt = at.Add(time.Minute)
	lastTick := at
	return persistence.Snapshot{
		Simulation: domain.SimulationState{
			Lab:          domain.LabState{EnergyBase: 100, EnergyBaseAt: at},
			Portals:      []domain.Portal{portal},
			Planes:       planes,
			Observers:    domain.NewObserverRoster(10, at),
			NextPortalID: 2,
			NaturalSpawn: domain.NaturalSpawnState{Paused: true},
			LastTickAt:   &lastTick,
		},
		App: domain.AppState{Mode: domain.ModeTutorial},
	}
}

func websocketURL(serverURL string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http")
}

func dialRealtime(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, websocketURL(serverURL), nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close(websocket.StatusNormalClosure, "test complete") })
	return conn
}

func readSnapshot(t *testing.T, conn *websocket.Conn) transport.StateSnapshot {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var snapshot transport.StateSnapshot
	require.NoError(t, wsjson.Read(ctx, conn, &snapshot))
	return snapshot
}

func TestWebSocket_ImmediatelyReceivesCurrentSnapshot(t *testing.T) {
	at := testutil.BaseTime
	source := newFakeStateSource(realtimeSnapshot(at))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	got := readSnapshot(t, dialRealtime(t, server.URL))
	want, err := transport.BuildStateSnapshot(source.snapshot, at, config.Default())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWebSocket_TickBroadcastUsesRESTSnapshotSchema(t *testing.T) {
	at := testutil.BaseTime
	source := newFakeStateSource(realtimeSnapshot(at))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)
	conn := dialRealtime(t, server.URL)
	_ = readSnapshot(t, conn)

	next := realtimeSnapshot(at.Add(time.Second))
	next.Simulation.Lab.EnergyBase = 73
	source.replace(next)

	got := readSnapshot(t, conn)
	want, err := transport.BuildStateSnapshot(next, at.Add(time.Second), config.Default())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestWebSocket_DoesNotExposeHiddenFields(t *testing.T) {
	at := testutil.BaseTime
	source := newFakeStateSource(realtimeSnapshot(at))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)
	conn := dialRealtime(t, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var raw json.RawMessage
	require.NoError(t, wsjson.Read(ctx, conn, &raw))
	body := string(raw)
	for _, hidden := range []string{
		"energy_base", "energy_base_at", "decay_rate", "energy_lifetime",
		"instability_collapse_at", "spawn_due_at", "last_tick_at", "risk_score",
	} {
		require.NotContains(t, body, hidden)
	}
}

func TestHub_RemovesDisconnectedClient(t *testing.T) {
	source := newFakeStateSource(realtimeSnapshot(testutil.BaseTime))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)
	conn := dialRealtime(t, server.URL)
	_ = readSnapshot(t, conn)
	require.Eventually(t, func() bool { return hub.clientCount() == 1 }, time.Second, 10*time.Millisecond)

	require.NoError(t, conn.Close(websocket.StatusNormalClosure, "done"))
	require.Eventually(t, func() bool { return hub.clientCount() == 0 }, time.Second, 10*time.Millisecond)
}

func TestWebSocket_ReconnectGetsLatestSnapshot(t *testing.T) {
	at := testutil.BaseTime
	source := newFakeStateSource(realtimeSnapshot(at))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	first := dialRealtime(t, server.URL)
	_ = readSnapshot(t, first)
	require.NoError(t, first.Close(websocket.StatusNormalClosure, "reconnect"))
	require.Eventually(t, func() bool { return hub.clientCount() == 0 }, time.Second, 10*time.Millisecond)

	latest := realtimeSnapshot(at.Add(3 * time.Second))
	latest.Simulation.Lab.EnergyBase = 61
	source.replace(latest)
	got := readSnapshot(t, dialRealtime(t, server.URL))
	want, err := transport.BuildStateSnapshot(latest, at.Add(3*time.Second), config.Default())
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestHub_SlowClientDoesNotBlockFastClientOrManager(t *testing.T) {
	source := newFakeStateSource(realtimeSnapshot(testutil.BaseTime))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)

	slowCtx, cancelSlow := context.WithCancel(context.Background())
	fastCtx, cancelFast := context.WithCancel(context.Background())
	slow := &client{queue: make(chan queuedSnapshot, 1), cancel: cancelSlow}
	fast := &client{queue: make(chan queuedSnapshot, 1), cancel: cancelFast}
	require.NoError(t, hub.register(slowCtx, slow))
	require.NoError(t, hub.register(fastCtx, fast))
	t.Cleanup(func() {
		hub.unregister(slow)
		hub.unregister(fast)
	})
	<-slow.queue
	<-fast.queue

	hub.publish(queuedSnapshot{sequence: 100, view: transport.StateSnapshot{GeneratedAt: testutil.BaseTime}})
	done := make(chan struct{})
	go func() {
		hub.publish(queuedSnapshot{sequence: 101, view: transport.StateSnapshot{GeneratedAt: testutil.BaseTime.Add(time.Second)}})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("slow client blocked snapshot publication")
	}
	select {
	case got := <-fast.queue:
		require.Equal(t, uint64(101), got.sequence)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("fast client did not receive latest snapshot")
	}
}

func TestHub_CoalescesPendingSnapshots(t *testing.T) {
	source := newFakeStateSource(realtimeSnapshot(testutil.BaseTime))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	candidate := &client{queue: make(chan queuedSnapshot, 1), cancel: cancel}
	require.NoError(t, hub.register(ctx, candidate))
	t.Cleanup(func() { hub.unregister(candidate) })
	<-candidate.queue

	hub.publish(queuedSnapshot{sequence: 10, view: transport.StateSnapshot{GeneratedAt: testutil.BaseTime}})
	hub.publish(queuedSnapshot{sequence: 11, view: transport.StateSnapshot{GeneratedAt: testutil.BaseTime.Add(time.Second)}})
	require.Len(t, candidate.queue, 1)
	require.Equal(t, uint64(11), (<-candidate.queue).sequence)
}

func TestWebSocket_ConcurrentConnectBroadcastDisconnect(t *testing.T) {
	source := newFakeStateSource(realtimeSnapshot(testutil.BaseTime))
	hub, err := NewHub(source, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(hub)
	t.Cleanup(server.Close)

	const clients = 12
	start := make(chan struct{})
	errs := make(chan error, clients)
	var wg sync.WaitGroup
	for i := 0; i < clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			conn, _, err := websocket.Dial(ctx, websocketURL(server.URL), nil)
			if err != nil {
				errs <- err
				return
			}
			var snapshot transport.StateSnapshot
			if err := wsjson.Read(ctx, conn, &snapshot); err != nil {
				errs <- err
				_ = conn.CloseNow()
				return
			}
			if err := conn.Close(websocket.StatusNormalClosure, "done"); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	for i := 1; i <= 20; i++ {
		next := realtimeSnapshot(testutil.BaseTime.Add(time.Duration(i) * time.Second))
		next.Simulation.Lab.EnergyBase = 100 - i
		source.replace(next)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Eventually(t, func() bool { return hub.clientCount() == 0 }, time.Second, 10*time.Millisecond)
}
