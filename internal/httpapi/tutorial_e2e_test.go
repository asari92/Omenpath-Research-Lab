package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

type minimumCheckpointRandom struct{}

func (*minimumCheckpointRandom) IntInclusive(min, _ int) int       { return min }
func (*minimumCheckpointRandom) FloatRange(min, _ float64) float64 { return min }
func (*minimumCheckpointRandom) MarshalBinary() ([]byte, error)    { return []byte{}, nil }
func (*minimumCheckpointRandom) UnmarshalBinary([]byte) error      { return nil }

func openTutorialE2EStore(t *testing.T, path string, now time.Time) *persistence.Store {
	t.Helper()
	store, err := persistence.Open(context.Background(), path)
	require.NoError(t, err)
	require.NoError(t, store.Migrate(context.Background()))
	require.NoError(t, store.Bootstrap(context.Background(), now, config.Default()))
	return store
}

func persistTutorialPortal(t *testing.T, store *persistence.Store, snapshot persistence.Snapshot, profile domain.TutorialPortalProfile, step int, phase domain.TutorialPhase, now time.Time) persistence.Snapshot {
	t.Helper()
	before := snapshot
	portal, err := domain.NewTutorialPortal(profile, snapshot.Simulation.NextPortalID, 1, 1, now, config.Default())
	require.NoError(t, err)
	snapshot.Simulation.Portals = append(snapshot.Simulation.Portals, portal)
	snapshot.Simulation.NextPortalID++
	snapshot.App = domain.AppState{
		Mode: domain.ModeTutorial, TutorialStep: step, TutorialPhase: phase,
		TutorialPortalID: &portal.ID, TutorialPlaneID: &portal.DestinationPlaneID,
	}
	drafts, err := domain.EventsForStateTransition(before.Simulation, snapshot.Simulation, now, now, config.Default())
	require.NoError(t, err)
	_, err = store.Commit(context.Background(), snapshot, drafts)
	require.NoError(t, err)
	return snapshot
}

func TestTutorialAPI_CriticalRejectionPersistsProgressAndEvent(t *testing.T) {
	ctx := context.Background()
	base := testutil.BaseTime
	store := openTutorialE2EStore(t, filepath.Join(t.TempDir(), "critical.sqlite"), base)
	defer store.Close()
	snapshot, err := store.Load(ctx)
	require.NoError(t, err)
	snapshot = persistTutorialPortal(t, store, snapshot, domain.TutorialPortalStep5, 5, domain.TutorialPhaseNone, base)
	clock := testutil.NewFakeClock(base)
	manager, err := engine.NewLabManager(ctx, config.Default(), clock, &minimumCheckpointRandom{}, store)
	require.NoError(t, err)
	router, err := NewRouter(manager, config.Default())
	require.NoError(t, err)
	defer router.Close()
	server := httptest.NewServer(router)
	defer server.Close()
	conn := connectRouterWebSocket(t, server.URL)
	initial := receiveRouterSnapshot(t, conn)
	require.Equal(t, 5, initial.App.TutorialStep)

	response, err := server.Client().Post(server.URL+"/api/portals/1/send-observer", "application/json", strings.NewReader(`{}`))
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, response.StatusCode)
	require.NoError(t, response.Body.Close())
	updated := receiveRouterSnapshot(t, conn)
	require.Equal(t, 6, updated.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseSendReplacement, updated.App.TutorialPhase)

	persisted, err := store.Load(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, persisted.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseSendReplacement, persisted.App.TutorialPhase)
	events, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, []domain.EventType{domain.EventActionRejected, domain.EventPortalOpened}, []domain.EventType{
		events[len(events)-2].EventType,
		events[len(events)-1].EventType,
	})
	require.Equal(t, snapshot.Simulation.Portals[0].ID, *events[len(events)-2].PortalID)
	require.Equal(t, *persisted.App.TutorialPortalID, *events[len(events)-1].PortalID)
	opened := 0
	for _, event := range events {
		if event.EventType == domain.EventPortalOpened && event.PortalID != nil && *event.PortalID == *persisted.App.TutorialPortalID {
			opened++
		}
	}
	require.Equal(t, 1, opened)
}

func TestTutorialWebSocket_ReconnectShowsPersistedStepAndPhase(t *testing.T) {
	ctx := context.Background()
	base := testutil.BaseTime
	path := filepath.Join(t.TempDir(), "reconnect.sqlite")
	store := openTutorialE2EStore(t, path, base)
	snapshot, err := store.Load(ctx)
	require.NoError(t, err)
	snapshot = persistTutorialPortal(t, store, snapshot, domain.TutorialPortalStep6Outbound, 6, domain.TutorialPhaseSendReplacement, base)
	portalCount := len(snapshot.Simulation.Portals)
	eventsBefore, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	manager, err := engine.NewLabManager(ctx, config.Default(), testutil.NewFakeClock(base), &minimumCheckpointRandom{}, store)
	require.NoError(t, err)
	router, err := NewRouter(manager, config.Default())
	require.NoError(t, err)
	server := httptest.NewServer(router)
	conn := connectRouterWebSocket(t, server.URL)
	first := receiveRouterSnapshot(t, conn)
	require.Equal(t, domain.TutorialPhaseSendReplacement, first.App.TutorialPhase)
	conn.CloseNow()
	server.Close()
	router.Close()
	require.NoError(t, store.Close())

	store = openTutorialE2EStore(t, path, base)
	defer store.Close()
	manager, err = engine.NewLabManager(ctx, config.Default(), testutil.NewFakeClock(base), &minimumCheckpointRandom{}, store)
	require.NoError(t, err)
	router, err = NewRouter(manager, config.Default())
	require.NoError(t, err)
	defer router.Close()
	server = httptest.NewServer(router)
	defer server.Close()
	conn = connectRouterWebSocket(t, server.URL)
	reconnected := receiveRouterSnapshot(t, conn)
	require.Equal(t, first.App, reconnected.App)
	loaded, err := store.Load(ctx)
	require.NoError(t, err)
	require.Len(t, loaded.Simulation.Portals, portalCount)
	eventsAfter, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, eventsBefore, eventsAfter)
}

func TestTutorial_FullColdStartRestartAndLiveJourney(t *testing.T) {
	ctx := context.Background()
	base := testutil.BaseTime
	path := filepath.Join(t.TempDir(), "journey.sqlite")
	store := openTutorialE2EStore(t, path, base)
	clock := testutil.NewFakeClock(base)
	manager, err := engine.NewLabManager(ctx, config.Default(), clock, &minimumCheckpointRandom{}, store)
	require.NoError(t, err)
	initial, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, initial.App.TutorialStep)
	require.Empty(t, initial.Simulation.Portals)

	require.NoError(t, manager.TutorialSignal(ctx, domain.TutorialSignalIntroCompleted, nil))
	step1, err := manager.State(ctx)
	require.NoError(t, err)
	require.NoError(t, manager.TutorialSignal(ctx, domain.TutorialSignalPortalDetailsOpened, step1.App.TutorialPortalID))
	clock.Advance(6 * time.Second)
	require.NoError(t, manager.Tick(ctx))
	step3, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, step3.App.TutorialStep)
	sendPortalID := *step3.App.TutorialPortalID
	require.NoError(t, manager.SendObserver(ctx, sendPortalID, true))
	step4, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, step4.App.TutorialStep)
	require.NoError(t, manager.Stabilize(ctx, *step4.App.TutorialPortalID))
	step5, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, step5.App.TutorialStep)
	require.ErrorIs(t, manager.SendObserver(ctx, *step5.App.TutorialPortalID, true), domain.ErrPortalCriticalRisk)

	beforeRestart, err := store.Load(ctx)
	require.NoError(t, err)
	eventsBeforeRestart, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, store.Close())
	store = openTutorialE2EStore(t, path, base)
	defer store.Close()
	manager, err = engine.NewLabManager(ctx, config.Default(), clock, &minimumCheckpointRandom{}, store)
	require.NoError(t, err)
	afterRestart, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, beforeRestart.App, afterRestart.App)
	require.Len(t, afterRestart.Simulation.Portals, len(beforeRestart.Simulation.Portals))
	eventsAfterRestart, err := store.ListEvents(ctx, nil)
	require.NoError(t, err)
	require.Equal(t, eventsBeforeRestart, eventsAfterRestart)

	clock.Advance(25 * time.Second)
	require.NoError(t, manager.Tick(ctx))
	recallReady, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.TutorialPhaseRecallReady, recallReady.App.TutorialPhase)
	require.NoError(t, manager.RecallObserver(ctx, *recallReady.App.TutorialPortalID, true))
	clock.Advance(5 * time.Second)
	require.NoError(t, manager.Tick(ctx))
	step8, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, 8, step8.App.TutorialStep)
	require.True(t, step8.Simulation.Planes[0].Explored)
	eventsAtStep8, err := manager.Events(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, eventsAtStep8)
	require.Equal(t, 8, step8.App.TutorialStep, "Event Log GET must stay read-only")
	require.NoError(t, manager.TutorialSignal(ctx, domain.TutorialSignalEventLogOpened, nil))
	beforeLive, err := manager.State(ctx)
	require.NoError(t, err)
	energyBeforeLive := beforeLive.Simulation.Lab.CurrentEnergy(clock.Now(), config.Default())
	eventCountBeforeLive := len(eventsAtStep8)
	require.NoError(t, manager.StartLive(ctx))
	live, err := manager.State(ctx)
	require.NoError(t, err)
	require.Equal(t, domain.ModeLive, live.App.Mode)
	require.Equal(t, 0, openPortalCount(live.Simulation.Portals))
	require.Equal(t, energyBeforeLive, live.Simulation.Lab.CurrentEnergy(clock.Now(), config.Default()))
	require.True(t, live.Simulation.Planes[0].Explored)
	require.Equal(t, domain.ObserverAvailable, live.Simulation.Observers[0].Status)
	require.False(t, live.Simulation.NaturalSpawn.Paused)
	require.NotNil(t, live.Simulation.NaturalSpawn.ScheduledAt)
	require.NotNil(t, live.Simulation.NaturalSpawn.DueAt)
	eventsAfterLive, err := manager.Events(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(eventsAfterLive), eventCountBeforeLive)
}

func openPortalCount(portals []domain.Portal) int {
	count := 0
	for _, portal := range portals {
		if portal.Status == domain.PortalStatusOpen {
			count++
		}
	}
	return count
}
