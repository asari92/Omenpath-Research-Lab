package transport

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

func transitSnapshot() persistence.Snapshot {
	now := testutil.BaseTime
	s := dtoSnapshot(now)
	first := &s.Simulation.Portals[0]
	first.ObserverFlow = domain.PortalFlowOutbound
	second := *first
	second.ID, second.SlotIndex, second.ObserverFlow = 2, 2, domain.PortalFlowInbound
	s.Simulation.Portals = append(s.Simulation.Portals, second)
	s.Simulation.NextPortalID = 3
	start, end, plane, p1, p2 := now.Add(-2*time.Second), now.Add(8*time.Second), int64(1), int64(1), int64(2)
	s.Simulation.Observers = []domain.Observer{
		{ID: 9, Status: domain.ObserverOutbound, ActivePortalID: &p1, PhaseStartedAt: &start, PhaseEndsAt: &end},
		{ID: 3, Status: domain.ObserverReturning, ActivePortalID: &p2, CurrentPlaneID: &plane, PhaseStartedAt: &start, PhaseEndsAt: &end},
	}
	return s
}

func jsonFields(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &result))
	return result
}

func TestObserverTransitProjection_StateDetailsSortedAndScoped(t *testing.T) {
	s := transitSnapshot()
	state, err := BuildStateSnapshot(s, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	fields := jsonFields(t, state)
	require.Contains(t, fields, "observer_transits")
	var transits []map[string]any
	require.NoError(t, json.Unmarshal(fields["observer_transits"], &transits))
	require.Len(t, transits, 2)
	require.Equal(t, float64(3), transits[0]["observer_id"])
	require.Equal(t, "RETURNING", transits[0]["direction"])
	require.Equal(t, "OUTBOUND", transits[1]["direction"])
	require.Equal(t, float64(8), transits[1]["remaining_seconds"])
	require.Equal(t, testutil.BaseTime.Add(-2*time.Second).Format(time.RFC3339), transits[1]["started_at"])
	require.Equal(t, testutil.BaseTime.Add(8*time.Second).Format(time.RFC3339), transits[1]["completes_at"])
	for i, id := range []int64{2, 1} {
		detail, err := BuildPortalDetails(s, id, nil, testutil.BaseTime, config.Default())
		require.NoError(t, err)
		var transit map[string]any
		require.NoError(t, json.Unmarshal(jsonFields(t, detail)["observer_transit"], &transit))
		require.Equal(t, transits[i], transit)
	}
}

func TestObserverTransitProjection_NoneAfterResolvedDeadline(t *testing.T) {
	s := dtoSnapshot(testutil.BaseTime)
	state, err := BuildStateSnapshot(s, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.JSONEq(t, `[]`, string(jsonFields(t, state)["observer_transits"]))
	detail, err := BuildPortalDetails(s, 1, nil, testutil.BaseTime, config.Default())
	require.NoError(t, err)
	require.JSONEq(t, `null`, string(jsonFields(t, detail)["observer_transit"]))
}

func TestObserverTransitProjection_RejectsMalformed(t *testing.T) {
	cases := map[string]func(*persistence.Snapshot){
		"missing portal": func(s *persistence.Snapshot) { s.Simulation.Observers[0].ActivePortalID = nil },
		"missing start":  func(s *persistence.Snapshot) { s.Simulation.Observers[0].PhaseStartedAt = nil },
		"missing end":    func(s *persistence.Snapshot) { s.Simulation.Observers[0].PhaseEndsAt = nil },
		"reversed phase": func(s *persistence.Snapshot) {
			s.Simulation.Observers[0].PhaseEndsAt = s.Simulation.Observers[0].PhaseStartedAt
		},
		"future start": func(s *persistence.Snapshot) {
			v := testutil.BaseTime.Add(time.Second)
			s.Simulation.Observers[0].PhaseStartedAt = &v
		},
		"unknown portal":     func(s *persistence.Snapshot) { v := int64(99); s.Simulation.Observers[0].ActivePortalID = &v },
		"outbound plane":     func(s *persistence.Snapshot) { v := int64(1); s.Simulation.Observers[0].CurrentPlaneID = &v },
		"return wrong plane": func(s *persistence.Snapshot) { v := int64(2); s.Simulation.Observers[1].CurrentPlaneID = &v },
		"wrong flow":         func(s *persistence.Snapshot) { s.Simulation.Portals[0].ObserverFlow = domain.PortalFlowInbound },
		"duplicate observer": func(s *persistence.Snapshot) { s.Simulation.Observers[1].ID = 9 },
		"duplicate portal transit": func(s *persistence.Snapshot) {
			s.Simulation.Observers = append(s.Simulation.Observers, s.Simulation.Observers[0])
			s.Simulation.Observers[2].ID = 10
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			s := transitSnapshot()
			change(&s)
			_, err := BuildStateSnapshot(s, testutil.BaseTime, config.Default())
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
			_, err = BuildPortalDetails(s, 1, nil, testutil.BaseTime, config.Default())
			require.ErrorIs(t, err, domain.ErrSimulationInvariant)
		})
	}
}
