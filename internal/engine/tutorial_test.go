package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
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

func TestTutorial_StateReadAtStep2ResolvesTimeWithoutCompletingCorridor(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalPortalDetailsOpened, manager.snapshot.App.TutorialPortalID))
	wantApp := cloneSnapshot(manager.snapshot).App
	wantPortalCount := len(manager.snapshot.Simulation.Portals)
	wantEvents := len(repo.events)
	manager.clock = testutil.NewFakeClock(testutil.BaseTime.Add(30 * time.Second))

	got, err := manager.State(context.Background())
	require.NoError(t, err)
	require.Equal(t, wantApp, got.App)
	require.Equal(t, 2, manager.snapshot.App.TutorialStep)
	require.Len(t, manager.snapshot.Simulation.Portals, wantPortalCount)
	require.Len(t, repo.events, wantEvents)
	require.Equal(t, 0, got.Simulation.Portals[0].CreaturesInside(manager.clock.Now(), manager.cfg))
	require.Equal(t, manager.clock.Now(), *got.Simulation.LastTickAt)
}

func TestTutorial_StateReadAtStep6ResolvesResearchWithoutCreatingReturnTarget(t *testing.T) {
	manager, repo := tutorialStep6ResearchManager(t)
	wantApp := cloneSnapshot(manager.snapshot).App
	wantPortalCount := len(manager.snapshot.Simulation.Portals)

	got, err := manager.State(context.Background())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverWaitingReturn, got.Simulation.Observers[0].Status)
	require.Equal(t, wantApp, got.App)
	require.Len(t, got.Simulation.Portals, wantPortalCount)
	require.Equal(t, []domain.EventType{domain.EventResearchCompleted}, eventTypes(repo.events))
}

func TestTutorial_StateReadAtStep7ResolvesReturnWithoutAdvancingToEventLog(t *testing.T) {
	manager, repo := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.NoError(t, manager.RecallObserver(context.Background(), *manager.snapshot.App.TutorialPortalID, true))
	wantApp := cloneSnapshot(manager.snapshot).App
	wantPortalCount := len(manager.snapshot.Simulation.Portals)
	wantEventCount := len(repo.events)
	manager.clock = testutil.NewFakeClock(*manager.snapshot.Simulation.Observers[0].PhaseEndsAt)

	got, err := manager.State(context.Background())
	require.NoError(t, err)
	require.Equal(t, domain.ObserverAvailable, got.Simulation.Observers[0].Status)
	require.True(t, got.Simulation.Planes[0].Explored)
	require.Equal(t, wantApp, got.App)
	require.Len(t, got.Simulation.Portals, wantPortalCount)
	require.Greater(t, len(repo.events), wantEventCount, "ordinary return/exploration events still persist")
}

func TestTutorial_PortalStateReadResolvesTerminalTargetWithoutRecreatingIt(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	oldID := *manager.snapshot.App.TutorialPortalID
	wantApp := cloneSnapshot(manager.snapshot).App
	manager.clock = testutil.NewFakeClock(manager.snapshot.Simulation.Portals[0].ScheduledCloseAt.Add(time.Second))

	got, history, err := manager.PortalState(context.Background(), oldID)
	require.NoError(t, err)
	require.Equal(t, domain.PortalStatusClosed, got.Simulation.Portals[0].Status)
	require.Equal(t, wantApp, got.App)
	require.Len(t, got.Simulation.Portals, 1)
	require.Equal(t, []domain.EventType{domain.EventPortalOpened, domain.EventPortalClosed}, eventTypes(repo.events))
	require.Equal(t, []domain.EventType{domain.EventPortalOpened, domain.EventPortalClosed}, eventTypes(history))
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

func TestTutorial_Step3UsesNormalSendAndTracksObserverPlane(t *testing.T) {
	manager, repo := tutorialManager(t, 0, testutil.BaseTime)
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalIntroCompleted, nil))
	require.NoError(t, manager.TutorialSignal(context.Background(), domain.TutorialSignalPortalDetailsOpened, manager.snapshot.App.TutorialPortalID))
	manager.clock = testutil.NewFakeClock(testutil.BaseTime.Add(30 * time.Second))
	require.NoError(t, manager.Tick(context.Background()))
	sendPortalID := *manager.snapshot.App.TutorialPortalID
	planeID := manager.snapshot.Simulation.Portals[0].DestinationPlaneID

	require.NoError(t, manager.SendObserver(context.Background(), sendPortalID, true))
	require.Equal(t, 4, manager.snapshot.App.TutorialStep)
	require.Equal(t, &planeID, manager.snapshot.App.TutorialPlaneID)
	require.Equal(t, int64(1), *manager.snapshot.App.TutorialObserverID)
	require.Equal(t, domain.ObserverOutbound, manager.snapshot.Simulation.Observers[0].Status)
	require.Equal(t, &sendPortalID, manager.snapshot.Simulation.Observers[0].ActivePortalID)
	require.NotEqual(t, sendPortalID, *manager.snapshot.App.TutorialPortalID)
	require.Contains(t, eventTypes(repo.events), domain.EventObserverDispatched)
	require.Equal(t, 100, manager.snapshot.Simulation.Lab.EnergyBase)
}

func TestTutorial_Step4SuccessfulStabilizeUsesNormalDebitEventsAndCreatesStep5Target(t *testing.T) {
	manager, repo := tutorialManager(t, 4, testutil.BaseTime)
	portal, err := domain.NewTutorialPortal(domain.TutorialPortalStep4, 1, 1, 1, testutil.BaseTime, manager.cfg)
	require.NoError(t, err)
	portalID, planeID := portal.ID, portal.DestinationPlaneID
	manager.snapshot.Simulation.Portals = []domain.Portal{portal}
	manager.snapshot.Simulation.NextPortalID = 2
	manager.snapshot.App.TutorialPortalID = &portalID
	manager.snapshot.App.TutorialPlaneID = &planeID
	repo.snapshot = cloneSnapshot(manager.snapshot)

	require.NoError(t, manager.Stabilize(context.Background(), portalID))
	require.Equal(t, 80, manager.snapshot.Simulation.Lab.EnergyBase)
	require.Equal(t, domain.PortalStable, manager.snapshot.Simulation.Portals[0].Stability)
	risk, ok := manager.snapshot.Simulation.Portals[0].RiskLevel(testutil.BaseTime, manager.cfg)
	require.True(t, ok)
	require.Contains(t, []domain.RiskLevel{domain.RiskLow, domain.RiskMedium}, risk)
	require.Equal(t, 5, manager.snapshot.App.TutorialStep)
	require.NotEqual(t, portalID, *manager.snapshot.App.TutorialPortalID)
	require.Equal(t, domain.RiskCritical, func() domain.RiskLevel {
		level, _ := manager.snapshot.Simulation.Portals[1].RiskLevel(testutil.BaseTime, manager.cfg)
		return level
	}())
	types := eventTypes(repo.events)
	require.Contains(t, types, domain.EventPortalStabilized)
	require.Contains(t, types, domain.EventRiskLevelChanged)
	require.Contains(t, types, domain.EventPortalOpened)
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
	require.Equal(t, []domain.EventType{domain.EventActionRejected, domain.EventPortalOpened}, eventTypes(repo.events))
	require.Equal(t, portal.ID, *repo.events[0].PortalID)
	require.Equal(t, *manager.snapshot.App.TutorialPortalID, *repo.events[1].PortalID)
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

func TestTutorial_Step6WrongSuccessfulSendRecreatesRecallTargetAndSurvivesRestart(t *testing.T) {
	manager, repo := tutorialStep6ResearchManager(t)
	require.NoError(t, manager.Tick(context.Background()))
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	oldTargetID := *manager.snapshot.App.TutorialPortalID
	trackedObserverID := *manager.snapshot.App.TutorialObserverID
	eventsBefore := len(repo.events)

	require.NoError(t, manager.SendObserver(context.Background(), oldTargetID, true))
	require.Equal(t, 6, manager.snapshot.App.TutorialStep)
	require.Equal(t, domain.TutorialPhaseRecallReady, manager.snapshot.App.TutorialPhase)
	require.Equal(t, trackedObserverID, *manager.snapshot.App.TutorialObserverID)
	require.NotEqual(t, oldTargetID, *manager.snapshot.App.TutorialPortalID)
	require.Equal(t, domain.PortalFlowOutbound, manager.snapshot.Simulation.Portals[0].ObserverFlow)
	require.Equal(t, domain.ObserverOutbound, manager.snapshot.Simulation.Observers[1].Status)
	require.Equal(t, oldTargetID, *manager.snapshot.Simulation.Observers[1].ActivePortalID)
	fresh, ok := tutorialTarget(&manager.snapshot)
	require.True(t, ok)
	require.Equal(t, int64(1), fresh.DestinationPlaneID)
	require.Equal(t, domain.PortalFlowNone, fresh.ObserverFlow)
	require.Equal(t, []domain.EventType{domain.EventObserverDispatched, domain.EventPortalOpened}, eventTypes(repo.events[eventsBefore:]))

	restarted, err := NewLabManager(context.Background(), manager.cfg, manager.clock, &lockedMinimumRandom{}, repo)
	require.NoError(t, err)
	got, err := restarted.State(context.Background())
	require.NoError(t, err)
	require.Equal(t, manager.snapshot.App, got.App)
	require.Equal(t, manager.snapshot.Simulation.Portals, got.Simulation.Portals)
	require.Equal(t, manager.snapshot.Simulation.Observers, got.Simulation.Observers)
	require.Len(t, repo.events, eventsBefore+2)
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
	portal.ObserverFlow = domain.PortalFlowInbound
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

func TestTutorialStart_IsIdempotent(t *testing.T) {
	manager, repo := tutorialManager(t, 4, testutil.BaseTime)
	want := cloneSnapshot(manager.snapshot)
	require.NoError(t, manager.StartTutorial(context.Background()))
	require.NoError(t, manager.StartTutorial(context.Background()))
	require.Equal(t, want, manager.snapshot)
	require.Empty(t, repo.commits)
}

func TestTutorialReset_RestoresEnergyObserversPlanesAndClearsHistory(t *testing.T) {
	manager, repo := tutorialLostReturnManager(t)
	manager.snapshot.Simulation.Lab.EnergyBase = 3
	manager.snapshot.Simulation.Planes[0].Explored = true
	exploredAt := testutil.BaseTime
	manager.snapshot.Simulation.Planes[0].ExploredAt = &exploredAt
	repo.events = []domain.Event{{ID: 1, EventType: domain.EventPortalOpened, Message: "old", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime}}
	require.NoError(t, manager.ResetTutorial(context.Background()))
	require.Equal(t, domain.ModeTutorial, manager.snapshot.App.Mode)
	require.Zero(t, manager.snapshot.App.TutorialStep)
	require.Equal(t, 100, manager.snapshot.Simulation.Lab.EnergyBase)
	require.Empty(t, manager.snapshot.Simulation.Portals)
	require.Empty(t, repo.events)
	for _, plane := range manager.snapshot.Simulation.Planes {
		require.False(t, plane.Explored)
	}
	for _, observer := range manager.snapshot.Simulation.Observers {
		require.Equal(t, domain.ObserverAvailable, observer.Status)
	}
}

func TestTutorialReset_TransactionFailureRollsBackEverything(t *testing.T) {
	manager, repo := tutorialLostReturnManager(t)
	want := cloneSnapshot(manager.snapshot)
	repo.resetErr = errors.New("forced reset failure")
	err := manager.ResetTutorial(context.Background())
	require.ErrorContains(t, err, "forced reset failure")
	require.Equal(t, want, manager.snapshot)
}

func TestStartLive_BeforeStep9IsRejected(t *testing.T) {
	manager, repo := tutorialManager(t, 8, testutil.BaseTime)
	err := manager.StartLive(context.Background())
	require.ErrorIs(t, err, ErrTutorialNotReady)
	require.Equal(t, domain.ModeTutorial, manager.snapshot.App.Mode)
	require.Equal(t, domain.EventActionRejected, repo.events[len(repo.events)-1].EventType)
}

func TestStartLive_RejectionPersistsRecreatedTerminalTargetEventExactlyOnce(t *testing.T) {
	tests := []struct {
		name    string
		step    int
		phase   domain.TutorialPhase
		profile domain.TutorialPortalProfile
	}{
		{name: "step1", step: 1, profile: domain.TutorialPortalStep1},
		{name: "step2", step: 2, profile: domain.TutorialPortalStep2},
		{name: "step6_recall_ready", step: 6, phase: domain.TutorialPhaseRecallReady, profile: domain.TutorialPortalStep6Return},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			base := testutil.BaseTime
			manager, repo := tutorialManager(t, tc.step, base)
			portal, err := domain.NewTutorialPortal(tc.profile, 1, 1, 1, base, manager.cfg)
			require.NoError(t, err)
			portal.ScheduledCloseAt = base.Add(time.Second)
			portalID, planeID := portal.ID, portal.DestinationPlaneID
			manager.snapshot.Simulation.Portals = []domain.Portal{portal}
			manager.snapshot.Simulation.NextPortalID = 2
			manager.snapshot.App.TutorialPortalID = &portalID
			manager.snapshot.App.TutorialPlaneID = &planeID
			manager.snapshot.App.TutorialPhase = tc.phase
			if tc.step == 6 {
				observerID := manager.snapshot.Simulation.Observers[0].ID
				manager.snapshot.Simulation.Observers[0].Status = domain.ObserverWaitingReturn
				manager.snapshot.Simulation.Observers[0].CurrentPlaneID = &planeID
				manager.snapshot.Simulation.Observers[0].PhaseStartedAt = &base
				manager.snapshot.Simulation.Observers[0].UpdatedAt = base
				manager.snapshot.App.TutorialObserverID = &observerID
			}
			repo.snapshot = cloneTestSnapshot(manager.snapshot)
			manager.clock = testutil.NewFakeClock(base.Add(2 * time.Second))

			err = manager.StartLive(context.Background())
			require.ErrorIs(t, err, ErrTutorialNotReady)
			require.Equal(t, tc.step, manager.snapshot.App.TutorialStep)
			require.NotEqual(t, portalID, *manager.snapshot.App.TutorialPortalID)
			require.Equal(t, []domain.EventType{
				domain.EventPortalClosed,
				domain.EventPortalOpened,
				domain.EventActionRejected,
			}, eventTypes(repo.events))
			freshID := *manager.snapshot.App.TutorialPortalID
			require.Equal(t, freshID, *repo.events[1].PortalID)

			restarted, restartErr := NewLabManager(context.Background(), manager.cfg, manager.clock, &lockedMinimumRandom{}, repo)
			require.NoError(t, restartErr)
			got, stateErr := restarted.State(context.Background())
			require.NoError(t, stateErr)
			require.Equal(t, freshID, *got.App.TutorialPortalID)
			require.Len(t, got.Simulation.Portals, 2)
			require.Len(t, repo.events, 3)
		})
	}
}

func TestStartTutorial_LiveRejectionResolvesDueLifecycleAtomically(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	due := base.Add(time.Second)
	snapshot.Simulation.NaturalSpawn.DueAt = &due
	repo := newFakeRepository(snapshot)
	rnd := &checkpointSequenceRandom{
		ints:   []int{0, 10, 0, 1},
		floats: []float64{10, .1, 1},
	}
	manager, err := NewLabManager(context.Background(), config.Default(), testutil.NewFakeClock(base.Add(2*time.Second)), rnd, repo)
	require.NoError(t, err)

	err = manager.StartTutorial(context.Background())
	require.ErrorIs(t, err, ErrTutorialNotReady)
	require.Equal(t, domain.ModeLive, manager.snapshot.App.Mode)
	require.Len(t, manager.snapshot.Simulation.Portals, 1)
	require.Equal(t, []domain.EventType{domain.EventPortalOpened, domain.EventActionRejected}, eventTypes(repo.events))
	require.Equal(t, 4, rnd.intAt)
	require.Equal(t, 3, rnd.floatAt)
}

func TestStartTutorial_LiveRejectionFailureRollsBackCatchupAndRandom(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	due := base.Add(time.Second)
	snapshot.Simulation.NaturalSpawn.DueAt = &due
	repo := newFakeRepository(snapshot)
	rnd := &checkpointSequenceRandom{
		ints:   []int{0, 10, 0, 1},
		floats: []float64{10, .1, 1},
	}
	manager, err := NewLabManager(context.Background(), config.Default(), testutil.NewFakeClock(base.Add(2*time.Second)), rnd, repo)
	require.NoError(t, err)
	want := cloneSnapshot(manager.snapshot)
	repo.commitErr = errors.New("forced rejection failure")

	err = manager.StartTutorial(context.Background())
	require.ErrorContains(t, err, "forced rejection failure")
	require.Equal(t, want, manager.snapshot)
	require.Zero(t, rnd.intAt)
	require.Zero(t, rnd.floatAt)
	require.Len(t, repo.attempts, 1)
	require.Len(t, repo.attempts[0].snapshot.Simulation.Portals, 1)

	repo.commitErr = nil
	err = manager.StartTutorial(context.Background())
	require.ErrorIs(t, err, ErrTutorialNotReady)
	require.Len(t, repo.attempts, 2)
	require.Equal(t, repo.attempts[0], repo.attempts[1])
}

func TestStartLive_PreservesEnergyEventsObserversAndExploration(t *testing.T) {
	manager, repo := tutorialManager(t, 9, testutil.BaseTime)
	manager.snapshot.Simulation.Lab.EnergyBase = 37
	manager.snapshot.Simulation.Observers[0].Status = domain.ObserverLost
	manager.snapshot.Simulation.Planes[0].Explored = true
	exploredAt := testutil.BaseTime
	manager.snapshot.Simulation.Planes[0].ExploredAt = &exploredAt
	repo.snapshot = cloneSnapshot(manager.snapshot)
	repo.events = []domain.Event{{ID: 1, EventType: domain.EventPlaneExplored, PlaneID: int64Pointer(1), Message: "explored", PayloadJSON: `{}`, CreatedAt: testutil.BaseTime}}
	require.NoError(t, manager.StartLive(context.Background()))
	require.Equal(t, domain.ModeLive, manager.snapshot.App.Mode)
	require.Equal(t, 37, manager.snapshot.Simulation.Lab.EnergyBase)
	require.Equal(t, domain.ObserverLost, manager.snapshot.Simulation.Observers[0].Status)
	require.True(t, manager.snapshot.Simulation.Planes[0].Explored)
	require.NotEmpty(t, repo.events)
}

func TestStartLive_ClosesTutorialPortalsForFreeAndStartsZeroOpen(t *testing.T) {
	manager, _ := tutorialManager(t, 9, testutil.BaseTime)
	portal, err := domain.NewTutorialPortal(domain.TutorialPortalStep1, 1, 1, 1, testutil.BaseTime, manager.cfg)
	require.NoError(t, err)
	manager.snapshot.Simulation.Portals = []domain.Portal{portal}
	manager.snapshot.Simulation.NextPortalID = 2
	manager.snapshot.Simulation.Lab.EnergyBase = 4
	require.NoError(t, manager.StartLive(context.Background()))
	require.Equal(t, 4, manager.snapshot.Simulation.Lab.EnergyBase)
	require.Equal(t, domain.PortalStatusClosed, manager.snapshot.Simulation.Portals[0].Status)
	require.Equal(t, 0, countOpen(manager.snapshot.Simulation.Portals))
}

func TestStartLive_SchedulesNaturalGenerator(t *testing.T) {
	manager, _ := tutorialManager(t, 9, testutil.BaseTime)
	require.NoError(t, manager.StartLive(context.Background()))
	require.False(t, manager.snapshot.Simulation.NaturalSpawn.Paused)
	require.NotNil(t, manager.snapshot.Simulation.NaturalSpawn.ScheduledAt)
	require.NotNil(t, manager.snapshot.Simulation.NaturalSpawn.DueAt)
}

func activeTransitTutorialManager(t *testing.T, status domain.ObserverStatus) (*LabManager, *fakeRepository) {
	t.Helper()
	manager, repo := tutorialManager(t, 9, testutil.BaseTime)
	portal, err := domain.NewTutorialPortal(domain.TutorialPortalStep6Return, 1, 1, 1, testutil.BaseTime, manager.cfg)
	require.NoError(t, err)
	portalID, planeID := portal.ID, portal.DestinationPlaneID
	observer := &manager.snapshot.Simulation.Observers[0]
	switch status {
	case domain.ObserverOutbound:
		portal.ObserverFlow = domain.PortalFlowOutbound
		require.NoError(t, observer.StartOutbound(testutil.BaseTime, portalID, &lockedMinimumRandom{}, manager.cfg))
	case domain.ObserverReturning:
		portal.ObserverFlow = domain.PortalFlowInbound
		observer.Status = domain.ObserverWaitingReturn
		observer.CurrentPlaneID = &planeID
		observer.PhaseStartedAt = &testutil.BaseTime
		observer.UpdatedAt = testutil.BaseTime
		require.NoError(t, observer.StartReturning(testutil.BaseTime, portalID, &lockedMinimumRandom{}, manager.cfg))
	default:
		t.Fatalf("unsupported transit status %q", status)
	}
	manager.snapshot.Simulation.Portals = []domain.Portal{portal}
	manager.snapshot.Simulation.NextPortalID = 2
	repo.snapshot = cloneTestSnapshot(manager.snapshot)
	return manager, repo
}

func TestStartLive_ActiveTransitIsLostAtomicallyAndSurvivesRestart(t *testing.T) {
	for _, status := range []domain.ObserverStatus{domain.ObserverOutbound, domain.ObserverReturning} {
		t.Run(string(status), func(t *testing.T) {
			manager, repo := activeTransitTutorialManager(t, status)
			require.NoError(t, manager.StartLive(context.Background()))
			require.Equal(t, domain.ModeLive, manager.snapshot.App.Mode)
			require.Zero(t, countOpen(manager.snapshot.Simulation.Portals))
			observer := manager.snapshot.Simulation.Observers[0]
			require.Equal(t, domain.ObserverLost, observer.Status)
			require.Nil(t, observer.ActivePortalID)
			require.Nil(t, observer.CurrentPlaneID)
			require.Nil(t, observer.PhaseStartedAt)
			require.Nil(t, observer.PhaseEndsAt)
			require.Equal(t, []domain.EventType{domain.EventPortalClosed, domain.EventObserverLost}, eventTypes(repo.events))

			restarted, err := NewLabManager(context.Background(), manager.cfg, manager.clock, &lockedMinimumRandom{}, repo)
			require.NoError(t, err)
			got, err := restarted.State(context.Background())
			require.NoError(t, err)
			require.Equal(t, manager.snapshot, got)
			require.Len(t, repo.events, 2)
		})
	}
}

func TestStartLive_ActiveTransitCommitFailureRollsBackHandoff(t *testing.T) {
	manager, repo := activeTransitTutorialManager(t, domain.ObserverOutbound)
	want := cloneSnapshot(manager.snapshot)
	repo.commitErr = errors.New("forced live handoff failure")
	err := manager.StartLive(context.Background())
	require.ErrorContains(t, err, "forced live handoff failure")
	require.Equal(t, want, manager.snapshot)
	require.Empty(t, repo.events)

	repo.commitErr = nil
	require.NoError(t, manager.StartLive(context.Background()))
	require.Equal(t, domain.ObserverLost, manager.snapshot.Simulation.Observers[0].Status)
	require.Nil(t, manager.snapshot.Simulation.Observers[0].ActivePortalID)
}

func countOpen(portals []domain.Portal) int {
	count := 0
	for _, portal := range portals {
		if portal.Status == domain.PortalStatusOpen {
			count++
		}
	}
	return count
}

func eventTypes(events []domain.Event) []domain.EventType {
	result := make([]domain.EventType, len(events))
	for i := range events {
		result[i] = events[i].EventType
	}
	return result
}
