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
