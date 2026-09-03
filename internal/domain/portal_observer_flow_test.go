package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestNaturalPortal_ObserverFlowStartsNone(t *testing.T) {
	rnd := testutil.NewFakeRandom().
		QueueInt(60, 0).
		QueueFloat(100, 0.1, 0.9)

	portal := domain.NewNaturalPortal(11, 7, 1, testutil.BaseTime, config.Default(), rnd)

	require.Equal(t, domain.PortalFlowNone, portal.ObserverFlow)
}

func TestPortalFlow_DoesNotChangeOnRejectedFirstSend(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, domain.PortalFlowNone, portal.ObserverFlow)
}

func TestPortalFlow_DoesNotChangeOnRejectedFirstRecall(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, domain.PortalFlowNone, portal.ObserverFlow)
}

func TestPortalFlow_DoesNotResetAfterSuccessfulOutbound(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	end := testutil.BaseTime.Add(10 * time.Second)

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, end, config.Default()))

	require.Equal(t, domain.ObserverExploring, observers[0].Status)
	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
}

func TestPortalFlow_DoesNotResetAfterSuccessfulReturn(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	end := testutil.BaseTime.Add(10 * time.Second)

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, end, config.Default()))

	require.Equal(t, domain.ObserverAvailable, observers[0].Status)
	require.Equal(t, domain.PortalFlowInbound, portal.ObserverFlow)
}

func TestPortalFlow_DoesNotResetAfterObserverLost(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	closedAt := testutil.BaseTime.Add(5 * time.Second)
	require.NoError(t, portal.Close(closedAt, false, config.Default()))
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, closedAt, config.Default()))

	require.Equal(t, domain.ObserverLost, observers[0].Status)
	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
}

func TestPortalFlow_AllowsNextOutboundAfterPreviousTransitEnds(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{
		domain.NewObserver(1, testutil.BaseTime),
		domain.NewObserver(2, testutil.BaseTime),
	}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	now := testutil.BaseTime.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, now, config.Default()))

	id, err := domain.SendObserver(&portal, &plane, observers, now, false, testutil.NewFakeRandom().QueueInt(11), config.Default())
	require.NoError(t, err)
	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverOutbound, observers[1].Status)
}

func TestPortalFlow_AllowsNextInboundAfterPreviousTransitEnds(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	first := waitingObserver(testutil.BaseTime.Add(-2 * time.Minute))
	first.ID = 1
	second := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	second.ID = 2
	observers := []domain.Observer{first, second}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	now := testutil.BaseTime.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, now, config.Default()))

	id, err := domain.RecallObserver(&portal, &plane, observers, now, false, testutil.NewFakeRandom().QueueInt(11), config.Default())
	require.NoError(t, err)
	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverReturning, observers[1].Status)
}

func TestPortalFlow_RejectsRecallAfterOutboundTransitEnds(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	available := domain.NewObserver(1, testutil.BaseTime)
	waiting := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	waiting.ID = 2
	observers := []domain.Observer{available, waiting}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	now := testutil.BaseTime.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, now, config.Default()))

	_, err = domain.RecallObserver(&portal, &plane, observers, now, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
}

func TestPortalFlow_RejectsSendAfterInboundTransitEnds(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.NoError(t, err)
	now := testutil.BaseTime.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&observers[0], &plane, &portal, now, config.Default()))

	_, err = domain.SendObserver(&portal, &plane, observers, now, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
}

func TestPortalBusy_IsScopedToPortalID(t *testing.T) {
	end := testutil.BaseTime.Add(10 * time.Second)
	observer := outboundObserver(testutil.BaseTime, end)

	index, busy, err := domain.ActiveTransitObserverIndex([]domain.Observer{observer}, 12, testutil.BaseTime)

	require.NoError(t, err)
	require.False(t, busy)
	require.Equal(t, -1, index)
}

func TestPortalBusy_DoesNotBlockAnotherPortalToSamePlane(t *testing.T) {
	firstPortal := stage4Portal()
	secondPortal := stage4Portal()
	secondPortal.ID = 12
	secondPortal.SlotIndex = 2
	plane := stage4Plane()
	busy := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	available := domain.NewObserver(2, testutil.BaseTime)
	observers := []domain.Observer{busy, available}

	id, err := domain.SendObserver(&secondPortal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverOutbound, observers[1].Status)
	require.Equal(t, domain.PortalFlowNone, firstPortal.ObserverFlow)
	require.Equal(t, domain.PortalFlowOutbound, secondPortal.ObserverFlow)
}
