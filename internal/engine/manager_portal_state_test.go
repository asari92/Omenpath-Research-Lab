package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestManagerPortalState_ResolvesSnapshotAndHistoryAtOneBoundary(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	portal := managerPortal(1, 1, base)
	portal.ScheduledCloseAt = base.Add(time.Second)
	snapshot = withManagerPortal(snapshot, portal)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))

	got, history, err := manager.PortalState(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, got.Simulation.Portals[0].Status)
	require.NotNil(t, got.Simulation.LastTickAt)
	require.Equal(t, base.Add(2*time.Second), *got.Simulation.LastTickAt)
	types := make([]domain.EventType, len(history))
	for i := range history {
		types[i] = history[i].EventType
	}
	require.Equal(t, []domain.EventType{domain.EventPortalClosed}, types)
	require.NotNil(t, repo.lastFilter)
	require.Equal(t, int64(1), *repo.lastFilter)
}
