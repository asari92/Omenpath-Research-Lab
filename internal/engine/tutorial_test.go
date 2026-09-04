package engine

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func tutorialManager(t *testing.T, step int, now time.Time) (*LabManager, *fakeRepository) {
	t.Helper()
	snapshot := managerSnapshot(now)
	snapshot.App = domain.AppState{Mode: domain.ModeTutorial, TutorialStep: step}
	snapshot.Simulation.NaturalSpawn = domain.NaturalSpawnState{Paused: true}
	manager, repo := newTestManager(t, snapshot, now)
	return manager, repo
}

func TestTutorial_TicksDoNotAdvanceStep0OrCreatePortals(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, 0, manager.snapshot.App.TutorialStep)
	require.Empty(t, manager.snapshot.Simulation.Portals)
	require.True(t, manager.snapshot.Simulation.NaturalSpawn.Paused)
	require.Empty(t, repo.events)
}

func TestTutorial_IntroSignalAtomicallyCreatesOneStep1TargetAndEvent(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	require.Equal(t, 1, manager.snapshot.App.TutorialStep)
	require.Len(t, manager.snapshot.Simulation.Portals, 1)
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	require.Equal(t, manager.snapshot.Simulation.Portals[0].ID, *manager.snapshot.App.TutorialPortalID)
	require.Len(t, repo.events, 1)
	require.Equal(t, domain.EventPortalOpened, repo.events[0].EventType)
}

func TestTutorial_IntroSignalReplayIsRejectedWithoutDuplicate(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	err := manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil)
	require.ErrorIs(t, err, ErrInvalidTutorialSignal)
	require.Len(t, manager.snapshot.Simulation.Portals, 1)
	require.Equal(t, []domain.EventType{domain.EventPortalOpened, domain.EventActionRejected}, eventTypes(repo.events))
}

func TestTutorial_DetailsSignalRequiresMatchingTargetID(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	wrong := *manager.snapshot.App.TutorialPortalID + 1
	require.ErrorIs(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalPortalDetailsOpened, &wrong), ErrInvalidTutorialSignal)
	require.Equal(t, 1, manager.snapshot.App.TutorialStep)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalPortalDetailsOpened, manager.snapshot.App.TutorialPortalID))
	require.Equal(t, 2, manager.snapshot.App.TutorialStep)
}

func TestTutorial_ReadsNeverAdvanceProgress(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	for i := 0; i < 2; i++ {
		_, err := manager.State(context.Background())
		require.NoError(t, err)
	}
	require.Equal(t, 0, manager.snapshot.App.TutorialStep)
}

func TestTutorial_TickNeverSpawnsNaturalPortal(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	due := testutil.BaseTime
	manager.snapshot.Simulation.NaturalSpawn = domain.NaturalSpawnState{ScheduledAt: &due, DueAt: &due}
	require.NoError(t, manager.Tick(context.Background()))
	require.Empty(t, manager.snapshot.Simulation.Portals)
	require.True(t, manager.snapshot.Simulation.NaturalSpawn.Paused)
}

func TestTutorial_Step2AdvancesOnlyWhenCreaturesReachZero(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalPortalDetailsOpened, manager.snapshot.App.TutorialPortalID))
	require.Equal(t, 2, manager.snapshot.App.TutorialStep)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, 2, manager.snapshot.App.TutorialStep)
	manager.clock = testutil.NewFakeClock(testutil.BaseTime.Add(30 * time.Second))
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, 3, manager.snapshot.App.TutorialStep)
}

func TestTutorial_CriticalRejectedSendCommitsActionRejectedAndStep6(t *testing.T) {
	manager, repo := tutorialManager(t, 5, testutil.BaseTime)
	portal, err := domain.NewTutorialPortal(domain.TutorialPortalStep5, 1, 1, 1, testutil.BaseTime, manager.cfg)
	require.NoError(t, err)
	manager.snapshot.Simulation.Portals = []domain.Portal{portal}
	manager.snapshot.Simulation.NextPortalID = 2
	manager.snapshot.App.TutorialPortalID = &portal.ID
	repo.snapshot = cloneTestSnapshot(manager.snapshot)
	err = manager.SendObserver(context.Background(), portal.ID, true)
	require.ErrorIs(t, err, domain.ErrPortalCriticalRisk)
	require.Equal(t, 6, manager.snapshot.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseSendReplacement, manager.snapshot.App.TutorialPhase)
	require.Equal(t, domain.EventActionRejected, repo.events[len(repo.events)-1].EventType)
}

func TestTutorial_WrongReversibleActionKeepsStep(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	err := manager.TutorialSignal(context.Background(), domain.TutorialSignalEventLogOpened, nil)
	require.ErrorIs(t, err, ErrInvalidTutorialSignal)
	require.Equal(t, 0, manager.snapshot.App.TutorialStep)
}

func TestTutorial_TerminalTargetRecreatesEquivalentFreshID(t *testing.T) {
	manager, _ := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	oldID := *manager.snapshot.App.TutorialPortalID
	manager.clock = testutil.NewFakeClock(manager.snapshot.Simulation.Portals[0].ScheduledCloseAt.Add(time.Second))
	require.NoError(t, manager.Tick(context.Background()))
	require.NotEqual(t, oldID, *manager.snapshot.App.TutorialPortalID)
	require.Equal(t, domain.PortalStatusClosed, manager.snapshot.Simulation.Portals[0].Status)
	require.Equal(t, domain.PortalStatusOpen, manager.snapshot.Simulation.Portals[1].Status)
}

func eventTypes(events []domain.Event) []domain.EventType {
	result := make([]domain.EventType, len(events))
	for i := range events {
		result[i] = events[i].EventType
	}
	return result
}
