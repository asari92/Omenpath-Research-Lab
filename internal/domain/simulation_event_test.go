package domain_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestSimulationTick_ReplayHasNoEvents(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)
	first, err := state.ResolveTick(now, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.NotEmpty(t, first.Events)

	replay, err := state.ResolveTick(now, nil, config.Default())
	require.NoError(t, err)
	require.Empty(t, replay.Events)
}

func TestSimulationTick_EventOrderUsesSemanticTime(t *testing.T) {
	closing := tickNaturalClosePortal(1, 1, 4*time.Second)
	closing.ObserverFlow = domain.PortalFlowOutbound
	state := observerTickState(closing, tickOutboundObserver(1, 8*time.Second))
	state.NextPortalID = 2
	state.NaturalSpawn = scheduledSpawn(testutil.BaseTime, 5*time.Second)
	now := testutil.BaseTime.Add(5 * time.Second)

	result, err := state.ResolveTick(now, naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.Equal(t, []domain.EventType{
		domain.EventPortalClosed,
		domain.EventObserverLost,
		domain.EventPortalOpened,
	}, eventTypes(result.Events))
	require.Equal(t, []time.Time{
		testutil.BaseTime.Add(4 * time.Second),
		testutil.BaseTime.Add(4 * time.Second),
		now,
	}, eventTimes(result.Events))
}

func TestSimulationTick_ReturnsEventDraftsWithoutPersistenceIDs(t *testing.T) {
	state := dueNaturalState(testutil.BaseTime)
	result, err := state.ResolveTick(testutil.BaseTime.Add(time.Second), naturalSpawnRandom(0, 10, 0, 2, 10, .1, .9), config.Default())
	require.NoError(t, err)
	require.NotEmpty(t, result.Events)
	_, hasID := reflect.TypeOf(result.Events[0]).FieldByName("ID")
	require.False(t, hasID)
}
