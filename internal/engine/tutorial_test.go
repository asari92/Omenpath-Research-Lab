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

func tutorialStep6ResearchManager(t *testing.T) (*LabManager, *fakeRepository) {
	t.Helper()
	base := testutil.BaseTime
	manager, repo := tutorialManager(t, 6, base)
	planeID, observerID := int64(1), int64(1)
	started, ends := base, base.Add(manager.cfg.ResearchDuration)
	manager.snapshot.Simulation.Observers[0].Status = domain.ObserverExploring
	manager.snapshot.Simulation.Observers[0].CurrentPlaneID = &planeID
	manager.snapshot.Simulation.Observers[0].PhaseStartedAt = &started
	manager.snapshot.Simulation.Observers[0].PhaseEndsAt = &ends
	manager.snapshot.Simulation.Observers[0].UpdatedAt = base
	manager.snapshot.App.TutorialPhase = domain.TutorialPhaseWaitResearch
	manager.snapshot.App.TutorialPlaneID = &planeID
	manager.snapshot.App.TutorialObserverID = &observerID
	repo.snapshot = cloneTestSnapshot(manager.snapshot)
	manager.clock = testutil.NewFakeClock(ends)
	return manager, repo
}

func TestTutorial_EnterStep6DerivesWaitResearchPhase(t *testing.T) {
	manager, _ := tutorialStep6ResearchManager(t)
	require.Equal(t, domain.TutorialPhaseWaitResearch, manager.snapshot.App.TutorialPhase)
	require.Equal(t, domain.ObserverExploring, manager.snapshot.Simulation.Observers[0].Status)
}

func TestTutorial_Step6ResearchCompletionCreatesFreshSamePlanePortal(t *testing.T) {
	manager, _ := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	portal, ok := tutorialTarget(&manager.snapshot)
	require.True(t, ok)
	require.Equal(t, int64(1), portal.DestinationPlaneID)
	require.Equal(t, 0, portal.CreaturesInside(manager.clock.Now(), manager.cfg))
	require.Greater(t, portal.EffectiveLifetime(manager.clock.Now()), manager.cfg.ObserverTransitMax)
}

func TestTutorial_EnterStep6WaitingObserverCreatesSafeReturnPortal(t *testing.T) {
	manager, _ := tutorialStep6ResearchManager(t)
	manager.snapshot.Simulation.Observers[0].Status = domain.ObserverWaitingReturn
	manager.snapshot.Simulation.Observers[0].PhaseEndsAt = nil
	require.NoError(t, enterTutorialStep6(&manager.snapshot, manager.clock.Now(), manager.cfg))
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	portal, ok := tutorialTarget(&manager.snapshot)
	require.True(t, ok)
	require.Greater(t, portal.EffectiveLifetime(manager.clock.Now()), manager.cfg.ObserverTransitMax)
}

func TestTutorial_Step6RecallUsesLongestWaitingAndAdvancesStep7(t *testing.T) {
	manager, _ := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	portalID := *manager.snapshot.App.TutorialPortalID
	require.NoError(t, manager.RecallObserver(context.Background(), portalID, true))
	require.Equal(t, 7, manager.snapshot.App.TutorialStep)
	require.Equal(t, domain.ObserverReturning, manager.snapshot.Simulation.Observers[0].Status)
}

func TestTutorial_ReturnExploresPlaneOnlyAfterObserverReturned(t *testing.T) {
	manager, _ := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	require.NoError(t, manager.RecallObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true))
	require.False(t, manager.snapshot.Simulation.Planes[0].Explored)
	ends := *manager.snapshot.Simulation.Observers[0].PhaseEndsAt
	manager.clock = testutil.NewFakeClock(ends)
	require.NoError(t, manager.Tick(context.Background()))
	require.True(t, manager.snapshot.Simulation.Planes[0].Explored)
	require.Equal(t, 8, manager.snapshot.App.TutorialStep)
}

func tutorialLostReturnManager(t *testing.T) (*LabManager, *fakeRepository) {
	t.Helper()
	base := testutil.BaseTime
	manager, repo := tutorialManager(t, 7, base)
	portal, err := domain.NewTutorialPortal(domain.TutorialPortalStep6Return, 1, 1, 1, base, manager.cfg)
	require.NoError(t, err)
	portal.ScheduledCloseAt = base.Add(time.Second)
	planeID, portalID, observerID := int64(1), int64(1), int64(1)
	ends := base.Add(5 * time.Second)
	manager.snapshot.Simulation.Portals = []domain.Portal{portal}
	manager.snapshot.Simulation.NextPortalID = 2
	manager.snapshot.Simulation.Observers[0].Status = domain.ObserverReturning
	manager.snapshot.Simulation.Observers[0].CurrentPlaneID = &planeID
	manager.snapshot.Simulation.Observers[0].ActivePortalID = &portalID
	manager.snapshot.Simulation.Observers[0].PhaseStartedAt = &base
	manager.snapshot.Simulation.Observers[0].PhaseEndsAt = &ends
	manager.snapshot.Simulation.Observers[0].UpdatedAt = base
	manager.snapshot.App.TutorialPortalID = &portalID
	manager.snapshot.App.TutorialPlaneID = &planeID
	manager.snapshot.App.TutorialObserverID = &observerID
	repo.snapshot = cloneTestSnapshot(manager.snapshot)
	manager.clock = testutil.NewFakeClock(base.Add(2 * time.Second))
	return manager, repo
}

func TestTutorial_LostReturnUsesDifferentAvailableObserver(t *testing.T) {
	manager, _ := tutorialLostReturnManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.ObserverLost, manager.snapshot.Simulation.Observers[0].Status)
	require.Equal(t, 6, manager.snapshot.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseSendReplacement, manager.snapshot.App.TutorialPhase)
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	portalID := *manager.snapshot.App.TutorialPortalID
	require.NoError(t, manager.SendObserver(context.Background(), portalID, true))
	require.Equal(t, int64(2), *manager.snapshot.App.TutorialObserverID)
	require.Equal(t, domain.ObserverOutbound, manager.snapshot.Simulation.Observers[1].Status)
}

func TestTutorial_LostRetryNeverTeleportsObserverToPlane(t *testing.T) {
	manager, _ := tutorialLostReturnManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	require.NoError(t, manager.SendObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true))
	replacement := manager.snapshot.Simulation.Observers[1]
	require.Nil(t, replacement.CurrentPlaneID)
	require.Equal(t, domain.ObserverOutbound, replacement.Status)
}

func TestTutorial_LostRetryRequiresNormalSendResearchRecall(t *testing.T) {
	manager, _ := tutorialLostReturnManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.TutorialPhaseSendReplacement, manager.snapshot.App.TutorialPhase)
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	require.NoError(t, manager.SendObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true))
	require.Equal(t, domain.TutorialPhaseWaitResearch, manager.snapshot.App.TutorialPhase)
	ends := manager.snapshot.Simulation.Observers[1].PhaseEndsAt.Add(manager.cfg.ResearchDuration)
	manager.clock = testutil.NewFakeClock(ends)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	require.NoError(t, manager.RecallObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true))
	require.Equal(t, 7, manager.snapshot.App.TutorialStep)
}

func TestTutorial_EarlierTrackedLossEntersSendReplacementPhase(t *testing.T) {
	manager, _ := tutorialManager(t, 6, testutil.BaseTime)
	observerID, planeID := int64(1), int64(1)
	manager.snapshot.Simulation.Observers[0].Status = domain.ObserverLost
	manager.snapshot.App.TutorialObserverID = &observerID
	manager.snapshot.App.TutorialPlaneID = &planeID
	require.NoError(t, enterTutorialStep6(&manager.snapshot, testutil.BaseTime, manager.cfg))
	require.Equal(t, domain.TutorialPhaseSendReplacement, manager.snapshot.App.TutorialPhase)
}

func TestTutorial_NoReplacementObserverStaysRecoverableUntilReset(t *testing.T) {
	manager, _ := tutorialLostReturnManager(t)
	for i := 1; i < len(manager.snapshot.Simulation.Observers); i++ {
		manager.snapshot.Simulation.Observers[i].Status = domain.ObserverLost
	}
	require.NoError(t, manager.Tick(context.Background()))
	require.NotNil(t, manager.snapshot.App.TutorialPortalID)
	err := manager.SendObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true)
	require.ErrorIs(t, err, domain.ErrNoAvailableObserver)
	require.Equal(t, 6, manager.snapshot.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseSendReplacement, manager.snapshot.App.TutorialPhase)
}

func TestTutorial_EventLogGETDoesNotAdvanceStep8(t *testing.T) {
	manager, _ := tutorialManager(t, 8, testutil.BaseTime)
	_, err := manager.Events(context.Background())
	require.NoError(t, err)
	require.Equal(t, 8, manager.snapshot.App.TutorialStep)
}

func TestTutorial_EventLogSignalAdvancesStep8ToStep9(t *testing.T) {
	manager, _ := tutorialManager(t, 8, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalEventLogOpened, nil))
	require.Equal(t, 9, manager.snapshot.App.TutorialStep)
}

func TestTutorial_PhaseAndTargetRestartDoesNotDuplicatePortalOrEvents(t *testing.T) {
	manager, repo := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	restarted, err := NewLabManager(context.Background(), manager.cfg, manager.clock, &lockedMinimumRandom{}, repo)
	require.NoError(t, err)
	beforePortals, beforeEvents := len(restarted.snapshot.Simulation.Portals), len(repo.events)
	require.NoError(t, restarted.Tick(context.Background()))
	require.Len(t, restarted.snapshot.Simulation.Portals, beforePortals)
	require.Len(t, repo.events, beforeEvents)
}

func eventTypes(events []domain.Event) []domain.EventType {
	result := make([]domain.EventType, len(events))
	for i := range events {
		result[i] = events[i].EventType
	}
	return result
}
