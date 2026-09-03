package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabEmergency_OnlyCollapsedTransitionActivatesOverride(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	natural := testutil.NewPortalBuilder().Stable().Energy(100).Decay(0.1).TTL(5 * time.Second).Build()

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &natural, testutil.BaseTime.Add(5*time.Second), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Nil(t, lab.LeylineOverrideUntil)

	collapsing := energyCollapsePortal(10 * time.Second)
	changed, err = domain.ResolvePortalLifecycleWithLabEmergency(&lab, &collapsing, testutil.BaseTime.Add(10*time.Second), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.NotNil(t, lab.LeylineOverrideUntil)
}

func TestLabEmergency_ClosedPortalNeverActivatesOverride(t *testing.T) {
	cfg := config.Default()
	lab := labAt(75, testutil.BaseTime)
	portal := stage4Portal()
	require.NoError(t, portal.Close(testutil.BaseTime, false, cfg))
	labBefore := lab

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime, cfg)

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
}

func TestLabEmergency_AlreadyTerminalPortalIsIdempotent(t *testing.T) {
	cfg := config.Default()
	portal := energyCollapsePortal(5 * time.Second)
	_, err := portal.ResolveLifecycle(testutil.BaseTime.Add(5 * time.Second))
	require.NoError(t, err)
	lab := labAt(75, testutil.BaseTime)
	labBefore := lab
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(10*time.Second), cfg)

	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}

func TestLabEmergency_InvalidTransitionLeavesLabAndPortalUnchanged(t *testing.T) {
	cfg := config.Default()
	lab := labAt(40, testutil.BaseTime.Add(20*time.Second))
	portal := energyCollapsePortal(10 * time.Second)
	labBefore := lab
	portalBefore := portal

	changed, err := domain.ResolvePortalLifecycleWithLabEmergency(&lab, &portal, testutil.BaseTime.Add(20*time.Second), cfg)

	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.False(t, changed)
	require.Equal(t, labBefore, lab)
	require.Equal(t, portalBefore, portal)
}

func TestLabEmergency_OverrideActiveQueryIsPure(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab

	require.True(t, lab.LeylineOverrideActive(testutil.BaseTime.Add(time.Second), cfg))
	require.Equal(t, snapshot, lab)
}

func TestLabEmergency_OrdinaryCostsRemainAfterExpiry(t *testing.T) {
	cfg := config.Default()
	now := testutil.BaseTime.Add(cfg.EmergencyDuration)

	t.Run("close", func(t *testing.T) {
		lab := overrideLab(0, testutil.BaseTime, cfg)
		portal := stage4Portal()
		plane := stage4Plane()

		require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg))
		require.Equal(t, 15, lab.EnergyBase)
	})

	t.Run("stabilize", func(t *testing.T) {
		lab := overrideLab(0, testutil.BaseTime, cfg)
		portal := unstablePortal(50)

		require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg))
		require.Zero(t, lab.EnergyBase)
	})
}

func TestLabEmergency_SendAndRecallRemainZeroCost(t *testing.T) {
	cfg := config.Default()

	t.Run("send", func(t *testing.T) {
		lab := overrideLab(0, testutil.BaseTime, cfg)
		snapshot := lab
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), cfg)
		require.NoError(t, err)
		require.Equal(t, snapshot, lab)
	})

	t.Run("recall", func(t *testing.T) {
		lab := overrideLab(0, testutil.BaseTime, cfg)
		snapshot := lab
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

		_, err := domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), cfg)
		require.NoError(t, err)
		require.Equal(t, snapshot, lab)
	})
}

func TestLabEmergency_CloseFailureDoesNotConsumeRegeneratedEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab
	portal := stage4Portal()
	portal.CreaturesInitial = 1
	plane := stage4Plane()
	now := testutil.BaseTime.Add(time.Second)

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, cfg)

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, snapshot, lab)
	require.Equal(t, 1, lab.CurrentEnergy(now, cfg))
}

func TestLabEmergency_StabilizeFailureDoesNotConsumeRegeneratedEnergy(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	snapshot := lab
	portal := stage4Portal()
	now := testutil.BaseTime.Add(5 * time.Second)

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg)

	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
	require.Equal(t, snapshot, lab)
	require.Equal(t, 5, lab.CurrentEnergy(now, cfg))
}

func TestLabEmergency_LabAndPortalEnergyRemainIndependent(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := unstablePortal(50)
	now := testutil.BaseTime.Add(5 * time.Second)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, cfg))
	require.Equal(t, 5, lab.CurrentEnergy(now, cfg))
	require.InDelta(t, 62.5, portal.EnergyBase, 1e-9)
}

func TestLabEmergency_DoesNotMutateObserverUnlessCloseSucceeds(t *testing.T) {
	cfg := config.Default()
	lab := overrideLab(0, testutil.BaseTime, cfg)
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}
	snapshot := append([]domain.Observer(nil), observers...)

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, cfg)

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, snapshot, observers)
}

func TestLabEmergency_NoExtractionPortalBehavior(t *testing.T) {
	cfg := config.Default()
	cfg.LabRegenPerSec = cfg.ExtractionCost
	lab := overrideLab(0, testutil.BaseTime, cfg)
	var portals []domain.Portal

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime.Add(time.Second), cfg.ExtractionCost, cfg))
	require.Empty(t, portals)
}
