package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
)

var tutorialTestTime = time.Date(2026, 9, 5, 1, 0, 0, 0, time.UTC)

func TestTutorial_Step1PreparedPortalHasSafeCorridorAndSendProperties(t *testing.T) {
	now := tutorialTestTime
	portal, err := NewTutorialPortal(TutorialPortalStep1, 1, 1, 1, now, config.Default())
	require.NoError(t, err)
	require.Equal(t, PortalStable, portal.Stability)
	require.Greater(t, portal.CreaturesInside(now, config.Default()), 0)
	clearance := time.Duration(portal.CreaturesInside(now, config.Default())) * config.Default().CreatureTransit
	require.Greater(t, portal.EffectiveLifetime(now), clearance+config.Default().ObserverTransitMax)
}

func TestTutorial_Step4PreparedPortalSatisfiesStabilizeContract(t *testing.T) {
	now := tutorialTestTime
	portal, err := NewTutorialPortal(TutorialPortalStep4, 2, 2, 1, now, config.Default())
	require.NoError(t, err)
	before, ok := portal.RiskLevel(now, config.Default())
	require.True(t, ok)
	require.Contains(t, []RiskLevel{RiskHigh, RiskCritical}, before)
	require.Equal(t, PortalUnstable, portal.Stability)
	require.LessOrEqual(t, portal.CurrentEnergy(now), config.Default().StabilizeMaxStartEnergy)
	require.NoError(t, portal.Stabilize(now, config.Default()))
	after, ok := portal.RiskLevel(now, config.Default())
	require.True(t, ok)
	require.Contains(t, []RiskLevel{RiskLow, RiskMedium}, after)
}

func TestTutorial_StabilizeAllowsHighOrCriticalToMediumOrLow(t *testing.T) {
	for _, profile := range []TutorialPortalProfile{TutorialPortalStep4High, TutorialPortalStep4Critical} {
		portal, err := NewTutorialPortal(profile, 10, 1, 1, tutorialTestTime, config.Default())
		require.NoError(t, err)
		before, _ := portal.RiskLevel(tutorialTestTime, config.Default())
		require.Contains(t, []RiskLevel{RiskHigh, RiskCritical}, before)
		require.NoError(t, portal.Stabilize(tutorialTestTime, config.Default()))
		after, _ := portal.RiskLevel(tutorialTestTime, config.Default())
		require.Contains(t, []RiskLevel{RiskLow, RiskMedium}, after)
	}
}

func TestTutorial_Step5CriticalPortalEndsByNaturalCloseNotCollapse(t *testing.T) {
	now := tutorialTestTime
	portal, err := NewTutorialPortal(TutorialPortalStep5, 3, 3, 1, now, config.Default())
	require.NoError(t, err)
	risk, ok := portal.RiskLevel(now, config.Default())
	require.True(t, ok)
	require.Equal(t, RiskCritical, risk)
	depletion, ok := portal.EnergyDepletionAt()
	require.True(t, ok)
	require.False(t, depletion.Before(portal.ScheduledCloseAt))
	require.NoError(t, func() error { _, err := portal.ResolveLifecycle(portal.ScheduledCloseAt); return err }())
	require.Equal(t, PortalStatusClosed, portal.Status)
	require.Equal(t, TerminationNaturalClose, portal.TerminationReason)
}

func TestExpectedTutorialAction_UsesStepAndPhase(t *testing.T) {
	tests := []struct {
		app  AppState
		want TutorialExpectedAction
	}{
		{AppState{Mode: ModeTutorial, TutorialStep: 0}, TutorialActionCompleteIntro},
		{AppState{Mode: ModeTutorial, TutorialStep: 1}, TutorialActionOpenDetails},
		{AppState{Mode: ModeTutorial, TutorialStep: 2}, TutorialActionWaitCorridor},
		{AppState{Mode: ModeTutorial, TutorialStep: 6, TutorialPhase: TutorialPhaseSendReplacement}, TutorialActionSend},
		{AppState{Mode: ModeTutorial, TutorialStep: 6, TutorialPhase: TutorialPhaseWaitResearch}, TutorialActionWaitResearch},
		{AppState{Mode: ModeTutorial, TutorialStep: 6, TutorialPhase: TutorialPhaseRecallReady}, TutorialActionRecall},
		{AppState{Mode: ModeTutorial, TutorialStep: 9}, TutorialActionStartLive},
	}
	for _, tc := range tests {
		got, err := ExpectedTutorialAction(tc.app)
		require.NoError(t, err)
		require.Equal(t, tc.want, got)
	}
}
