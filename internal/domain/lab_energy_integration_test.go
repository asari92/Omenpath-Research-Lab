package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabEnergyCommands_InsufficientPaidActionDoesNotCallDomainMutation(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	portal := stage4Portal()
	snapshot := portal

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())

	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
	require.Equal(t, snapshot, portal)
}

func TestLabEnergyCommands_DomainFailureDoesNotDebit(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())

	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
	require.Equal(t, snapshot, lab)
}

func TestLabEnergyCommands_ConfirmationFailureDoesNotDebit(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

	err := domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, snapshot, lab)
}

func TestLabEnergyCommands_ZeroCostActionsPreserveBaselineAtCap(t *testing.T) {
	t.Run("send", func(t *testing.T) {
		lab := labAt(config.Default().LabEnergyMax, testutil.BaseTime)
		snapshot := lab
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime.Add(time.Second), false, testutil.NewFakeRandom().QueueInt(10), config.Default())
		require.NoError(t, err)
		require.Equal(t, snapshot, lab)
	})

	t.Run("recall", func(t *testing.T) {
		lab := labAt(config.Default().LabEnergyMax, testutil.BaseTime)
		snapshot := lab
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

		_, err := domain.RecallObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime.Add(time.Second), false, testutil.NewFakeRandom().QueueInt(10), config.Default())
		require.NoError(t, err)
		require.Equal(t, snapshot, lab)
	})
}

func TestLabEnergyCommands_PaidActionSpendsRegeneratedEnergyOnce(t *testing.T) {
	lab := labAt(4, testutil.BaseTime)
	portal := stage4Portal()
	plane := stage4Plane()
	now := testutil.BaseTime.Add(3 * time.Second)

	require.NoError(t, domain.ClosePortalWithLabEnergy(&lab, &portal, &plane, nil, now, false, config.Default()))
	require.Equal(t, 2, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
	require.Equal(t, 3, lab.CurrentEnergy(now.Add(time.Second), config.Default()))
}

func TestLabEnergyCommands_LabAndPortalEnergyRemainIndependent(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default()))
	require.Equal(t, 20, lab.EnergyBase)
	require.InDelta(t, 65.0, portal.EnergyBase, 1e-9)
}

func TestLabEnergyCommands_DoNotMutateOverrideDeadline(t *testing.T) {
	deadline := testutil.BaseTime.Add(time.Minute)
	lab := labAt(0, testutil.BaseTime)
	lab.LeylineOverrideUntil = &deadline
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
	require.Same(t, &deadline, lab.LeylineOverrideUntil)
	require.Equal(t, deadline, *lab.LeylineOverrideUntil)
}

func TestLabEnergyCommands_DoNotChangeStage4ErrorIdentity(t *testing.T) {
	t.Run("direction conflict", func(t *testing.T) {
		lab := labAt(0, testutil.BaseTime)
		portal := stage4Portal()
		portal.ObserverFlow = domain.PortalFlowInbound
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
		require.Equal(t, domain.ErrPortalDirectionConflict, err)
	})

	t.Run("busy portal", func(t *testing.T) {
		lab := labAt(0, testutil.BaseTime)
		portal := stage4Portal()
		portal.ObserverFlow = domain.PortalFlowOutbound
		plane := stage4Plane()
		observers := []domain.Observer{outboundObserver(testutil.BaseTime.Add(-time.Second), testutil.BaseTime.Add(10*time.Second))}

		_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
		require.Equal(t, domain.ErrPortalBusy, err)
	})
}

func TestLabEnergyCommands_DoNotChangeStage4RandomConsumption(t *testing.T) {
	t.Run("success consumes exactly once", func(t *testing.T) {
		lab := labAt(0, testutil.BaseTime)
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
		rnd := testutil.NewFakeRandom().QueueInt(10, 14)

		_, err := domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())
		require.NoError(t, err)
		require.Equal(t, 14, rnd.IntInclusive(5, 15))
	})

	t.Run("rejection consumes none", func(t *testing.T) {
		lab := labAt(0, testutil.BaseTime)
		portal := stage4Portal()
		portal.ObserverFlow = domain.PortalFlowInbound
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
		rnd := testutil.NewFakeRandom().QueueInt(13)

		_, _ = domain.SendObserverWithLabEnergy(&lab, &portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())
		require.Equal(t, 13, rnd.IntInclusive(5, 15))
	})
}

func TestLabEnergyCommands_RepeatedReadDoesNotWriteBaseline(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	now := testutil.BaseTime.Add(10 * time.Second)

	require.Equal(t, 50, lab.CurrentEnergy(now, config.Default()))
	require.Equal(t, 50, lab.CurrentEnergy(now, config.Default()))
	require.Equal(t, snapshot, lab)
}
