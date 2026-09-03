package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestObserverCommands_DoNotMutateOnStructuralInvariantError(t *testing.T) {
	t.Run("send mismatched destination", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		plane.ID = 99
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
		portalBefore := portal
		planeBefore := plane
		observersBefore := append([]domain.Observer(nil), observers...)

		_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.ErrorIs(t, err, domain.ErrObserverInvariant)
		require.Equal(t, portalBefore, portal)
		require.Equal(t, planeBefore, plane)
		require.Equal(t, observersBefore, observers)
	})

	t.Run("recall duplicate observer IDs", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		first := waitingObserver(testutil.BaseTime.Add(-time.Minute))
		first.ID = 1
		second := domain.NewObserver(1, testutil.BaseTime)
		observers := []domain.Observer{first, second}
		portalBefore := portal
		observersBefore := append([]domain.Observer(nil), observers...)

		_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.ErrorIs(t, err, domain.ErrObserverInvariant)
		require.Equal(t, portalBefore, portal)
		require.Equal(t, observersBefore, observers)
	})

	t.Run("close mismatched destination", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		plane.ID = 99
		observer := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
		observers := []domain.Observer{observer}
		portalBefore := portal
		observersBefore := append([]domain.Observer(nil), observers...)

		err := domain.ClosePortalWithObservers(&portal, &plane, observers, testutil.BaseTime.Add(5*time.Second), true, config.Default())

		require.ErrorIs(t, err, domain.ErrObserverInvariant)
		require.Equal(t, portalBefore, portal)
		require.Equal(t, observersBefore, observers)
	})

	t.Run("nil aggregate members", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		_, sendErr := domain.SendObserver(nil, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
		require.ErrorIs(t, sendErr, domain.ErrObserverInvariant)

		_, recallErr := domain.RecallObserver(&portal, nil, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
		require.ErrorIs(t, recallErr, domain.ErrObserverInvariant)

		closeErr := domain.ClosePortalWithObservers(nil, &plane, observers, testutil.BaseTime, false, config.Default())
		require.ErrorIs(t, closeErr, domain.ErrObserverInvariant)
	})
}

func TestObserverCommands_DoNotChangeUnselectedObservers(t *testing.T) {
	t.Run("send", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{
			domain.NewObserver(2, testutil.BaseTime),
			domain.NewObserver(1, testutil.BaseTime),
		}
		unselectedBefore := observers[0]

		_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.NoError(t, err)
		require.Equal(t, unselectedBefore, observers[0])
	})

	t.Run("recall", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		first := waitingObserver(testutil.BaseTime.Add(-time.Minute))
		first.ID = 1
		second := waitingObserver(testutil.BaseTime.Add(-30 * time.Second))
		second.ID = 2
		observers := []domain.Observer{first, second}
		unselectedBefore := observers[1]

		_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.NoError(t, err)
		require.Equal(t, unselectedBefore, observers[1])
	})
}

func TestObserverCommands_DoNotChangePlaneExplorationOnTransitStart(t *testing.T) {
	t.Run("send", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		before := plane
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.NoError(t, err)
		require.Equal(t, before, plane)
	})

	t.Run("recall", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		before := plane
		observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

		_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

		require.NoError(t, err)
		require.Equal(t, before, plane)
	})
}

func TestObserverCommands_DoNotConsumeRandomBeforeAllChecksPass(t *testing.T) {
	t.Run("send", func(t *testing.T) {
		portal := stage4Portal()
		portal.ObserverFlow = domain.PortalFlowInbound
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
		rnd := testutil.NewFakeRandom().QueueInt(13)

		_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

		require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
		require.Equal(t, 13, rnd.IntInclusive(5, 15))
	})

	t.Run("recall", func(t *testing.T) {
		portal := stage4Portal()
		portal.CreaturesInitial = 1
		plane := stage4Plane()
		observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
		rnd := testutil.NewFakeRandom().QueueInt(13)

		_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

		require.ErrorIs(t, err, domain.ErrPortalCreaturesPresent)
		require.Equal(t, 13, rnd.IntInclusive(5, 15))
	})
}

func TestObserverCommands_DifferentPortalsOperateIndependently(t *testing.T) {
	firstPortal := stage4Portal()
	secondPortal := stage4Portal()
	secondPortal.ID = 12
	secondPortal.SlotIndex = 2
	plane := stage4Plane()
	observers := []domain.Observer{
		domain.NewObserver(1, testutil.BaseTime),
		domain.NewObserver(2, testutil.BaseTime),
	}

	firstID, err := domain.SendObserver(&firstPortal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	secondID, err := domain.SendObserver(&secondPortal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(11), config.Default())

	require.NoError(t, err)
	require.Equal(t, int64(1), firstID)
	require.Equal(t, int64(2), secondID)
	require.Equal(t, firstPortal.ID, *observers[0].ActivePortalID)
	require.Equal(t, secondPortal.ID, *observers[1].ActivePortalID)
}

func TestObserverCommands_MultipleObserversMayRemainInSamePlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	first := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	first.ID = 1
	second := waitingObserver(testutil.BaseTime.Add(-30 * time.Second))
	second.ID = 2
	observers := []domain.Observer{first, second}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
	require.NotNil(t, observers[0].CurrentPlaneID)
	require.NotNil(t, observers[1].CurrentPlaneID)
	require.Equal(t, plane.ID, *observers[0].CurrentPlaneID)
	require.Equal(t, plane.ID, *observers[1].CurrentPlaneID)
}

func TestObserverCommands_Stage3TransitDeadlineRemainsFixed(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(10*time.Second), *observers[0].PhaseEndsAt)
	require.Equal(t, 14, rnd.IntInclusive(5, 15))
}

func TestObserverCommands_DoNotResetFlowDuringLifecycleCatchUp(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, testutil.BaseTime.Add(40*time.Second), config.Default()))

	require.Equal(t, domain.ObserverWaitingReturn, observers[0].Status)
	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
}
