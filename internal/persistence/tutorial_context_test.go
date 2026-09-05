package persistence

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
)

func TestTutorialContext_RoundTripsStepPhaseAndTargetsAcrossRestart(t *testing.T) {
	ctx := context.Background()
	store := openBootstrappedStore(t)
	now := time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC)
	snapshot := completeSnapshot(now)
	portalID, planeID, observerID := int64(7), int64(1), int64(1)
	snapshot.App = domain.AppState{
		Mode: domain.ModeTutorial, TutorialStep: 6, TutorialPhase: domain.TutorialPhaseRecallReady,
		TutorialPortalID: &portalID, TutorialPlaneID: &planeID, TutorialObserverID: &observerID,
	}
	_, err := testLab(t, store).Commit(ctx, snapshot, nil)
	require.NoError(t, err)

	loaded, err := testLab(t, store).Load(ctx)
	require.NoError(t, err)
	require.Equal(t, snapshot.App, loaded.App)
	require.NoError(t, store.Migrate(ctx), "migration must remain idempotent")
	var migrations int
	require.NoError(t, store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = 2`).Scan(&migrations))
	require.Equal(t, 1, migrations)
}
