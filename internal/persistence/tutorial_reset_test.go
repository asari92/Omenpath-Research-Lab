package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
)

func resetSnapshot(now time.Time) Snapshot {
	snapshot := completeSnapshot(now)
	snapshot.Simulation.Portals = []domain.Portal{}
	snapshot.Simulation.NextPortalID = 1
	snapshot.Simulation.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	snapshot.Simulation.LastTickAt = nil
	snapshot.Simulation.Lab = domain.LabState{EnergyBase: 100, EnergyBaseAt: now}
	for i := range snapshot.Simulation.Planes {
		snapshot.Simulation.Planes[i].Explored = false
		snapshot.Simulation.Planes[i].ExploredAt = nil
	}
	snapshot.Simulation.Observers = domain.NewObserverRoster(10, now)
	snapshot.App = domain.AppState{Mode: domain.ModeTutorial}
	return snapshot
}

func TestTutorialReset_RestoresEnergyObserversPlanesAndClearsHistory(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 5, 2, 0, 0, 0, time.UTC)
	before := completeSnapshot(now)
	portalID := before.Simulation.Portals[0].ID
	_, err := store.Commit(ctx, before, []domain.EventDraft{eventDraft(now, domain.EventPortalOpened, &portalID)})
	require.NoError(t, err)
	want := resetSnapshot(now.Add(time.Minute))

	require.NoError(t, store.ResetTutorial(ctx, want))
	got, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, want, got)
	events, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, events)
}

func TestTutorialReset_TransactionFailureRollsBackEverything(t *testing.T) {
	ctx := context.Background()
	store := openMigratedStore(t)
	now := time.Date(2026, 9, 5, 2, 10, 0, 0, time.UTC)
	before := completeSnapshot(now)
	portalID := before.Simulation.Portals[0].ID
	_, err := store.Commit(ctx, before, []domain.EventDraft{eventDraft(now, domain.EventPortalOpened, &portalID)})
	require.NoError(t, err)
	_, err = store.db.ExecContext(ctx, `CREATE TRIGGER fail_tutorial_reset BEFORE DELETE ON portals
		BEGIN SELECT RAISE(ABORT, 'forced reset failure'); END`)
	require.NoError(t, err)

	err = store.ResetTutorial(ctx, resetSnapshot(now.Add(time.Minute)))
	require.Error(t, err)
	got, loadErr := store.Load(ctx)
	require.NoError(t, loadErr)
	require.Equal(t, before, got)
	events, listErr := store.ListEvents(ctx, nil)
	require.NoError(t, listErr)
	require.Len(t, events, 1)
}
