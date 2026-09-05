package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

type transitManager struct{ *realtimeManager }

func (m transitManager) PortalState(ctx context.Context, _ int64) (persistence.Snapshot, []domain.Event, error) {
	s, e := m.State(ctx)
	return s, nil, e
}

func TestTransit_RESTDetailsAndWebSocketAgree(t *testing.T) {
	m := transitManager{newRealtimeManager()}
	p := &m.snapshot.Simulation.Portals[0]
	p.ObserverFlow = domain.PortalFlowOutbound
	start, end, id := testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(9*time.Second), p.ID
	m.snapshot.Simulation.Observers[0] = domain.Observer{ID: 1, Status: domain.ObserverOutbound, ActivePortalID: &id, PhaseStartedAt: &start, PhaseEndsAt: &end}
	router, err := newManagerRouter(m, config.Default())
	require.NoError(t, err)
	t.Cleanup(router.Close)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	fetch := func(path string) map[string]json.RawMessage {
		response, err := http.Get(server.URL + path)
		require.NoError(t, err)
		defer response.Body.Close()
		require.Equal(t, 200, response.StatusCode)
		var v map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(response.Body).Decode(&v))
		return v
	}
	state := fetch("/api/state")
	details := fetch("/api/portals/1")
	require.Contains(t, state, "observer_transits")
	require.Contains(t, details, "observer_transit")
	var items []json.RawMessage
	require.NoError(t, json.Unmarshal(state["observer_transits"], &items))
	require.Len(t, items, 1)
	require.JSONEq(t, string(items[0]), string(details["observer_transit"]))
	conn := connectRouterWebSocket(t, server.URL)
	ws := receiveRouterSnapshot(t, conn)
	payload, err := json.Marshal(ws)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &fields))
	require.JSONEq(t, string(state["observer_transits"]), string(fields["observer_transits"]))
}
