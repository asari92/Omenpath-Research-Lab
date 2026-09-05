package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/transport"
)

func TestSessions_WebSocketBroadcastIsolationOwnTicksAndLifetimeLease(t *testing.T) {
	service, registry, c := middlewareFixture(t)
	router, err := NewRouter(service, registry, config.Default(), false)
	require.NoError(t, err)
	server := httptest.NewServer(router)
	defer server.Close()
	ca := sessionRequest(t, router, "GET", "/api/state", "", nil).Result().Cookies()[0]
	cb := sessionRequest(t, router, "GET", "/api/state", "", nil).Result().Cookies()[0]
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	dial := func(cookie *http.Cookie) *websocket.Conn {
		header := http.Header{}
		header.Set("Cookie", cookie.String())
		conn, response, err := websocket.Dial(ctx, realtimeURL(server.URL), &websocket.DialOptions{HTTPHeader: header})
		require.NoError(t, err)
		require.Empty(t, response.Cookies())
		return conn
	}
	a, b := dial(ca), dial(cb)
	defer a.CloseNow()
	defer b.CloseNow()
	receiveRouterSnapshot(t, a)
	receiveRouterSnapshot(t, b)
	bUpdates := make(chan transport.StateSnapshot, 1)
	bErrors := make(chan error, 1)
	go func() {
		var snapshot transport.StateSnapshot
		err := wsjson.Read(ctx, b, &snapshot)
		if err != nil {
			bErrors <- err
			return
		}
		bUpdates <- snapshot
	}()
	response := sessionRequest(t, router, "POST", "/api/tutorial/signal", `{"signal":"TUTORIAL_INTRO_COMPLETED"}`, ca)
	require.Equal(t, 200, response.Code)
	changed := receiveRouterSnapshot(t, a)
	require.NotNil(t, changed.Slots[0].Portal)
	select {
	case snapshot := <-bUpdates:
		t.Fatalf("A action leaked to B: %+v", snapshot)
	case err := <-bErrors:
		t.Fatal(err)
	case <-time.After(40 * time.Millisecond):
	}
	c.Advance(time.Second)
	require.NoError(t, registry.TickAll(context.Background()))
	select {
	case snapshot := <-bUpdates:
		require.Nil(t, snapshot.Slots[0].Portal)
		require.Equal(t, c.Now(), snapshot.GeneratedAt)
	case err := <-bErrors:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("B did not receive its own tick")
	}
	resolved, err := service.Resolve(context.Background(), ca.Value)
	require.NoError(t, err)
	deleted, err := registry.DeleteIfIdle(resolved.LabID, func() error { t.Error("live WebSocket lease lost"); return nil })
	require.NoError(t, err)
	require.False(t, deleted)
	require.NoError(t, a.CloseNow())
	require.Eventually(t, func() bool {
		idle, err := registry.DeleteIfIdle(resolved.LabID, func() error { return nil })
		return err == nil && idle
	}, time.Second, time.Millisecond)
}
