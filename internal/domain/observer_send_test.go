package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func stage4Portal() domain.Portal {
	portal := testutil.NewPortalBuilder().
		Stable().
		Energy(100).
		Decay(0.1).
		TTL(120 * time.Second).
		Build()
	portal.ID = 11
	portal.DestinationPlaneID = 7
	return portal
}

func stage4Plane() domain.Plane {
	return domain.Plane{ID: 7, Name: "Innistrad"}
}

func sendObserver(t *testing.T, portal *domain.Portal, plane *domain.Plane, observers []domain.Observer, now time.Time, rnd *testutil.FakeRandom) int64 {
	t.Helper()
	id, err := domain.SendObserver(portal, plane, observers, now, false, rnd, config.Default())
	require.NoError(t, err)
	return id
}

func TestSendObserver_SelectsLowestAvailableObserver(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{
		domain.NewObserver(8, testutil.BaseTime),
		domain.NewObserver(2, testutil.BaseTime),
		domain.NewObserver(5, testutil.BaseTime),
	}

	id := sendObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverOutbound, observers[1].Status)
	require.Equal(t, domain.ObserverAvailable, observers[0].Status)
	require.Equal(t, domain.ObserverAvailable, observers[2].Status)
}

func TestSendObserver_StartsOutboundTransit(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	now := testutil.BaseTime.Add(3 * time.Second)

	sendObserver(t, &portal, &plane, observers, now, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.ObserverOutbound, observers[0].Status)
	require.NotNil(t, observers[0].ActivePortalID)
	require.Equal(t, portal.ID, *observers[0].ActivePortalID)
	require.Equal(t, now, *observers[0].PhaseStartedAt)
	require.Equal(t, now.Add(10*time.Second), *observers[0].PhaseEndsAt)
}

func TestSendObserver_ReturnsSelectedObserverID(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(42, testutil.BaseTime)}

	id := sendObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(42), id)
}

func TestSendObserver_FirstUseSetsOutboundFlow(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	now := testutil.BaseTime.Add(3 * time.Second)

	sendObserver(t, &portal, &plane, observers, now, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
	require.Equal(t, now, portal.UpdatedAt)
}

func TestSendObserver_ExistingOutboundFlowRemainsOutbound(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowOutbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendObserver(t, &portal, &plane, observers, testutil.BaseTime.Add(3*time.Second), testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, domain.PortalFlowOutbound, portal.ObserverFlow)
}

func TestSendObserver_UpdatesPortalOnlyWhenFlowFirstChanges(t *testing.T) {
	t.Run("first use", func(t *testing.T) {
		portal := stage4Portal()
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
		now := testutil.BaseTime.Add(3 * time.Second)

		sendObserver(t, &portal, &plane, observers, now, testutil.NewFakeRandom().QueueInt(10))

		require.Equal(t, now, portal.UpdatedAt)
	})

	t.Run("existing direction", func(t *testing.T) {
		portal := stage4Portal()
		portal.ObserverFlow = domain.PortalFlowOutbound
		previousUpdate := portal.UpdatedAt
		plane := stage4Plane()
		observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

		sendObserver(t, &portal, &plane, observers, testutil.BaseTime.Add(3*time.Second), testutil.NewFakeRandom().QueueInt(10))

		require.Equal(t, previousUpdate, portal.UpdatedAt)
	})
}

func TestSendObserver_DrawsTransitDurationExactlyOnce(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	rnd := testutil.NewFakeRandom().QueueInt(10, 14)

	sendObserver(t, &portal, &plane, observers, testutil.BaseTime, rnd)

	require.Equal(t, 14, rnd.IntInclusive(5, 15))
}

func TestSendObserver_DoesNotExplorePlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	before := plane
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, before, plane)
}

func TestSendObserver_AllowsAlreadyExploredPlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	exploredAt := testutil.BaseTime.Add(-time.Hour)
	plane.Explored = true
	plane.ExploredAt = &exploredAt
	before := plane
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	sendObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, before, plane)
	require.Equal(t, domain.ObserverOutbound, observers[0].Status)
}

func TestSendObserver_AllowsAnotherObserverInDestinationPlane(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	other := waitingObserver(testutil.BaseTime.Add(-time.Minute))
	other.ID = 9
	observers := []domain.Observer{other, domain.NewObserver(2, testutil.BaseTime)}

	id := sendObserver(t, &portal, &plane, observers, testutil.BaseTime, testutil.NewFakeRandom().QueueInt(10))

	require.Equal(t, int64(2), id)
	require.Equal(t, domain.ObserverWaitingReturn, observers[0].Status)
	require.Equal(t, domain.ObserverOutbound, observers[1].Status)
}

func TestSendObserver_RejectsClosedPortal(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestSendObserver_RejectsCollapsedPortal(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusCollapsed
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
}

func TestSendObserver_RejectsCriticalRisk(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(10 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalCriticalRisk)
}

func TestSendObserver_RejectsInboundFlow(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
}

func TestSendObserver_RejectsCreaturesInside(t *testing.T) {
	portal := stage4Portal()
	portal.CreaturesInitial = 1
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalCreaturesPresent)
}

func TestSendObserver_RejectsBusyPortal(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	busy := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{busy, domain.NewObserver(2, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalBusy)
}

func TestSendObserver_RejectsWhenNoObserverAvailable(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	observers := []domain.Observer{waitingObserver(testutil.BaseTime.Add(-time.Minute))}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrNoAvailableObserver)
}

func TestSendObserver_UnstableRequiresConfirmation(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	portalBefore := portal
	observersBefore := append([]domain.Observer(nil), observers...)

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrConfirmationRequired)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, observersBefore, observers)
}

func TestSendObserver_UnstableConfirmedStartsTransit(t *testing.T) {
	portal := stage4Portal()
	portal.Stability = domain.PortalUnstable
	hidden := testutil.BaseTime.Add(time.Minute)
	portal.InstabilityCollapseAt = &hidden
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	id, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, true, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
	require.Equal(t, int64(1), id)
	require.Equal(t, domain.ObserverOutbound, observers[0].Status)
}

func TestSendObserver_HighRiskStablePortalNeedsNoConfirmation(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(14 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestSendObserver_LowEnergyDoesNotCreateSeparateWarning(t *testing.T) {
	portal := stage4Portal()
	portal.EnergyBase = 5
	portal.EnergyDecayRate = 0.1
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestSendObserver_LowRemainingTimeDoesNotCreateSeparateWarning(t *testing.T) {
	portal := stage4Portal()
	portal.ScheduledCloseAt = testutil.BaseTime.Add(14 * time.Second)
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestSendObserver_ExploredPlaneDoesNotCreateSeparateWarning(t *testing.T) {
	portal := stage4Portal()
	plane := stage4Plane()
	plane.Explored = true
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom().QueueInt(10), config.Default())

	require.NoError(t, err)
}

func TestSendObserver_RejectionIsAtomic(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	portalBefore := portal
	planeBefore := plane
	observersBefore := append([]domain.Observer(nil), observers...)

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, portalBefore, portal)
	require.Equal(t, planeBefore, plane)
	require.Equal(t, observersBefore, observers)
}

func TestSendObserver_RejectionDoesNotConsumeRandom(t *testing.T) {
	portal := stage4Portal()
	portal.ObserverFlow = domain.PortalFlowInbound
	plane := stage4Plane()
	observers := []domain.Observer{domain.NewObserver(1, testutil.BaseTime)}
	rnd := testutil.NewFakeRandom().QueueInt(13)

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, rnd, config.Default())

	require.ErrorIs(t, err, domain.ErrPortalDirectionConflict)
	require.Equal(t, 13, rnd.IntInclusive(5, 15))
}

func TestSendObserver_UsesDocumentedErrorPrecedence(t *testing.T) {
	portal := stage4Portal()
	portal.Status = domain.PortalStatusClosed
	portal.ObserverFlow = domain.PortalFlowInbound
	portal.CreaturesInitial = 1
	portal.Stability = domain.PortalUnstable
	plane := stage4Plane()
	busy := outboundObserver(testutil.BaseTime, testutil.BaseTime.Add(10*time.Second))
	observers := []domain.Observer{busy}

	_, err := domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)

	portal.Status = domain.PortalStatusOpen
	portal.ScheduledCloseAt = testutil.BaseTime.Add(10 * time.Second)
	_, err = domain.SendObserver(&portal, &plane, observers, testutil.BaseTime, false, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrPortalCriticalRisk)
}
