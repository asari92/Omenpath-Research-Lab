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
