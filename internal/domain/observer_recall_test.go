package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func recallObserver(t *testing.T, portal *domain.Portal, plane *domain.Plane, observers []domain.Observer, now time.Time, rnd *testutil.FakeRandom) int64 {
	t.Helper()
	id, err := domain.RecallObserver(portal, plane, observers, now, false, rnd, config.Default())
	require.NoError(t, err)
	return id
}

func TestRecallObserver_SelectsLongestWaitingInDestination(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	newer := waitingObserver(testutil.BaseTime.Add(-20 * time.Second))
	newer.ID = 2
	older := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	older.ID = 8
	observers := []domain.Observer{newer, older}

	id := recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(8), id)
	require.Equal(t, domain.ObserverWaitingReturn, observers[0].Status)
	require.Equal(t, domain.ObserverReturning, observers[1].Status)
}

func TestRecallObserver_BreaksWaitingTieByLowestID(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	since := testutil.BaseTime.Add(-time.Minute)
	higherID := waitingObserver(since)
	higherID.ID = 8
	lowerID := waitingObserver(since)
	lowerID.ID = 2
	observers := []domain.Observer{higherID, lowerID}

	id := recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverReturning, observers[1].Status)
}

func TestRecallObserver_IgnoresWaitingObserversInOtherPlanes(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	otherPlaneID := int64(99)
	other := waitingObserver(testutil.BaseTime.Add(-time.Hour))
	other.ID = 1
	other.CurrentPlaneID = &otherPlaneID
	matching := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	matching.ID = 9
	observers := []domain.Observer{other, matching}

	id := recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(9), id)
	require.Equal(t, domain.ObserverWaitingReturn, observers[0].Status)
	require.Equal(t, domain.ObserverReturning, observers[1].Status)
}

func TestRecallObserver_StartsReturningTransit(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	observers := []domain.Observer{observer}
	now := testutil.BaseTime

	recallObserver(t, &portal, &plane, observers, now, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.ObserverReturning, observers[0].Status)
	require.NotNil(t, observers[0].ActivePortalID)
	require.Equal(t, portal.ID, *observers[0].ActivePortalID)
	require.Equal(t, now, *observers[0].PhaseStartedAt)
	require.Equal(t, now.Add(10*time.Second), *observers[0].PhaseEndsAt)
}

func TestRecallObserver_PreservesCurrentPlaneDuringTransit(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.NotNil(t, observers[0].CurrentPlaneID)
	require.Equal(t, plane.ID, *observers[0].CurrentPlaneID)
}

func TestRecallObserver_ReturnsSelectedObserverID(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observer := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	observer.ID = 42
	observers := []domain.Observer{observer}

	id := recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(42), id)
}

func TestRecallObserver_FirstUseSetsInboundFlow(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	now := testutil.BaseTime.Add(3 * time.Second)

	recallObserver(t, &portal, &plane, observers, now, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.PortalFlowInbound, portal.ObserverFlow)
	require.Equal(t, now, portal.UpdatedAt)
}

func TestRecallObserver_ExistingInboundFlowRemainsInbound(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	previousUpdate := portal.UpdatedAt
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	recallObserver(t, &portal, &plane, observers, testutil.BaseTime.Add(3*time.Second), testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.PortalFlowInbound, portal.ObserverFlow)
	require.Equal(t, previousUpdate, portal.UpdatedAt)
}

func TestRecallObserver_DrawsFreshTransitDurationExactlyOnce(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	recallObserver(t, &portal, &plane, observers, testutil.BaseTime, rnd)

	require.Equal(t, 14, rnd.IntInclusive(5, 15))
}

func TestRecallObserver_LeavesOtherWaitingObserversUnchanged(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	selected := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	selected.ID = 1
	other := waitingObserver(testutil.BaseTime.Add(-30 * time.Second))
	other.ID = 2
	observers := []domain.Observer{selected, other}
	otherBefore := observers[1]

	recallObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, otherBefore, observers[1])
}

func TestRecallObserver_RejectsClosedPortal(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestRecallObserver_RejectsCollapsedPortal(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusCollapsed
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestRecallObserver_RejectsCriticalRisk(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(10 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalCriticalRisk)
}

func TestRecallObserver_RejectsOutboundFlow(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
}

func TestRecallObserver_RejectsCreaturesInside(t *testing.T) {
	portal := stage4Portal()
	portal.CreaturesInitial = 1
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalCreaturesPresent)
}

func TestRecallObserver_RejectsBusyPortal(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	busy := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	waiting := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	waiting.ID = 2
	observers := []domain.Observer{busy, waiting}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalBusy)
}

func TestRecallObserver_RejectsWhenDestinationHasNoWaitingObserver(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrNoWaitingObserver)
}

func TestRecallObserver_DoesNotUseWaitingObserverFromOtherPlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	otherPlaneID := int64(99)
	observer := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	observer.CurrentPlaneID = &otherPlaneID
	observers := []domain.Observer{observer}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrNoWaitingObserver)
	require.Equal(t, domain.ObserverWaitingReturn, observers[0].Status)
}

func TestRecallObserver_UnstableRequiresConfirmation(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
}

func TestRecallObserver_UnstableConfirmedStartsTransit(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	id, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, true, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
	require.Equal(t, int64(1), id)
	require.Equal(t, domain.ObserverReturning, observers[0].Status)
}

func TestRecallObserver_HighRiskStablePortalNeedsNoConfirmation(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(14 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestRecallObserver_LowEnergyDoesNotCreateSeparateWarning(t *testing.T) {
	portal := stage4Portal()
	portal.EnergyBase = 5
	portal.EnergyDecayRate = 0.1
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestRecallObserver_LowRemainingTimeDoesNotCreateSeparateWarning(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(14 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestRecallObserver_RejectionIsAtomic(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	portalBefore := portal
	planeBefore := plane
	observersBefore := append([]domain.Observer(nil), observers...)

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, planeBefore, plane)
	require.Equal(t, observersBefore, observers)
}

func TestRecallObserver_RejectionDoesNotConsumeRandom(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}
	rnd := testutil.NewFakeRandom().QueueInt(13, 14)

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}

func TestRecallObserver_UsesDocumentedErrorPrecedence(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	portal.ObserverFlow = domain.PortalFlowOutbound
	portal.CreaturesInitial = 1
	portal.Stability = domain.PortalUnstable
	plane := stage4Plane()
	busy := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	waiting := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	waiting.ID = 2
	observers := []domain.Observer{busy, waiting}

	_, err := domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)

	portal.Status = domain.PortalStatusOpen
	portal.ScheduledCloseAt = testutil.BaseTime.Add(10 * time.Second)
	_, err = domain.RecallObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalCriticalRisk)
}
