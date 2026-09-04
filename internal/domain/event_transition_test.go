package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestEventsForTransition_NaturalOpenAndDedicatedExtractionOpen(t *testing.T) {
	t.Run("natural", func(t *testing.T) {
		portal := testutil.NewPortalBuilder().Build()
		events := transitionEvents(t, domain.SimulationState{}, domain.SimulationState{Portals: []domain.Portal{portal}}, testutil.BaseTime, testutil.BaseTime)
		require.Equal(t, []domain.EventType{domain.EventPortalOpened}, eventTypes(events))
		require.Equal(t, portal.OpenedAt, events[0].CreatedAt)
	})

	t.Run("extraction", func(t *testing.T) {
		portal := tickExtractionPortal(1, 1, 1, testutil.BaseTime)
		events := transitionEvents(t, domain.SimulationState{}, domain.SimulationState{Portals: []domain.Portal{portal}}, testutil.BaseTime, testutil.BaseTime)
		require.Equal(t, []domain.EventType{domain.EventExtractionPortalOpened}, eventTypes(events))
	})
}

func TestEventsForTransition_CloseAndCollapse(t *testing.T) {
	t.Run("close", func(t *testing.T) {
		beforePortal := testutil.NewPortalBuilder().Build()
		afterPortal := beforePortal
		closedAt := testutil.BaseTime.Add(4 * time.Second)
		makeTerminalPortal(&afterPortal, domain.PortalStatusClosed, domain.TerminationNaturalClose, closedAt)

		events := transitionEvents(t, stateWithPortals(beforePortal), stateWithPortals(afterPortal), testutil.BaseTime, closedAt)
		require.Equal(t, []domain.EventType{domain.EventPortalClosed}, eventTypes(events))
		require.Equal(t, closedAt, events[0].CreatedAt)
	})

	t.Run("collapse starts override", func(t *testing.T) {
		beforePortal := testutil.NewPortalBuilder().Build()
		afterPortal := beforePortal
		closedAt := testutil.BaseTime.Add(4 * time.Second)
		makeTerminalPortal(&afterPortal, domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, closedAt)
		deadline := closedAt.Add(config.Default().EmergencyDuration)
		before := stateWithPortals(beforePortal)
		after := stateWithPortals(afterPortal)
		after.Lab.LeylineOverrideUntil = &deadline

		events := transitionEvents(t, before, after, testutil.BaseTime, closedAt)
		require.Equal(t, []domain.EventType{domain.EventPortalCollapsed, domain.EventLeylineOverrideStarted}, eventTypes(events))
		require.NotNil(t, events[1].PortalID)
		require.Equal(t, afterPortal.ID, *events[1].PortalID)
	})
}

func TestEventsForTransition_OverrideRestartAndEndExactlyOnce(t *testing.T) {
	cfg := config.Default()
	oldDeadline := testutil.BaseTime.Add(10 * time.Second)
	beforePortal := testutil.NewPortalBuilder().Build()
	afterPortal := beforePortal
	collapseAt := testutil.BaseTime.Add(5 * time.Second)
	makeTerminalPortal(&afterPortal, domain.PortalStatusCollapsed, domain.TerminationEnergyDepleted, collapseAt)
	newDeadline := collapseAt.Add(cfg.EmergencyDuration)
	before := stateWithPortals(beforePortal)
	before.Lab.LeylineOverrideUntil = &oldDeadline
	after := stateWithPortals(afterPortal)
	after.Lab.LeylineOverrideUntil = &newDeadline

	restart := transitionEvents(t, before, after, testutil.BaseTime, collapseAt)
	require.Equal(t, []domain.EventType{domain.EventPortalCollapsed, domain.EventLeylineOverrideStarted}, eventTypes(restart))

	beforeEnd := domain.SimulationState{Lab: domain.LabState{LeylineOverrideUntil: &newDeadline}}
	afterEnd := beforeEnd
	ended := transitionEvents(t, beforeEnd, afterEnd, collapseAt, newDeadline)
	require.Equal(t, []domain.EventType{domain.EventLeylineOverrideEnded}, eventTypes(ended))
	require.Equal(t, newDeadline, ended[0].CreatedAt)

	replayed := transitionEvents(t, afterEnd, afterEnd, newDeadline, newDeadline.Add(time.Second))
	require.Empty(t, replayed)
}

func TestEventsForTransition_OutboundArrivalStartsResearch(t *testing.T) {
	portalID, planeID := int64(1), int64(2)
	arrivalAt := testutil.BaseTime.Add(5 * time.Second)
	startedAt := testutil.BaseTime
	beforeObserver := eventObserver(1, domain.ObserverOutbound, nil, &portalID, &startedAt, &arrivalAt)
	researchEndsAt := arrivalAt.Add(config.Default().ResearchDuration)
	afterObserver := eventObserver(1, domain.ObserverExploring, &planeID, nil, &arrivalAt, &researchEndsAt)

	events := transitionEvents(t, stateWithObservers(beforeObserver), stateWithObservers(afterObserver), testutil.BaseTime, arrivalAt)
	require.Equal(t, []domain.EventType{domain.EventObserverArrived, domain.EventResearchStarted}, eventTypes(events))
	for _, event := range events {
		require.Equal(t, arrivalAt, event.CreatedAt)
		require.Equal(t, int64(1), *event.ObserverID)
		require.Equal(t, planeID, *event.PlaneID)
	}
}

func TestEventsForTransition_LateTickReconstructsObserverPhases(t *testing.T) {
	portalID, planeID := int64(1), int64(2)
	arrivalAt := testutil.BaseTime.Add(5 * time.Second)
	startedAt := testutil.BaseTime
	beforeObserver := eventObserver(1, domain.ObserverOutbound, nil, &portalID, &startedAt, &arrivalAt)
	completedAt := arrivalAt.Add(config.Default().ResearchDuration)
	afterObserver := eventObserver(1, domain.ObserverWaitingReturn, &planeID, nil, &completedAt, nil)

	events := transitionEvents(t, stateWithObservers(beforeObserver), stateWithObservers(afterObserver), testutil.BaseTime, completedAt.Add(5*time.Second))
	require.Equal(t, []domain.EventType{
		domain.EventObserverArrived,
		domain.EventResearchStarted,
		domain.EventResearchCompleted,
	}, eventTypes(events))
	require.Equal(t, []time.Time{arrivalAt, arrivalAt, completedAt}, eventTimes(events))
}

func TestEventsForTransition_ReturnAndLoss(t *testing.T) {
	t.Run("return then plane explored", func(t *testing.T) {
		planeID, portalID := int64(1), int64(2)
		startedAt := testutil.BaseTime
		returnedAt := testutil.BaseTime.Add(5 * time.Second)
		beforeObserver := eventObserver(1, domain.ObserverReturning, &planeID, &portalID, &startedAt, &returnedAt)
		afterObserver := eventObserver(1, domain.ObserverAvailable, nil, nil, nil, nil)
		before := stateWithObservers(beforeObserver)
		before.Planes = []domain.Plane{{ID: planeID}}
		after := stateWithObservers(afterObserver)
		after.Planes = []domain.Plane{{ID: planeID, Explored: true, ExploredAt: &returnedAt}}

		events := transitionEvents(t, before, after, testutil.BaseTime, returnedAt)
		require.Equal(t, []domain.EventType{domain.EventObserverReturned, domain.EventPlaneExplored}, eventTypes(events))
	})

	t.Run("loss", func(t *testing.T) {
		portalID := int64(2)
		startedAt := testutil.BaseTime
		transitEndsAt := testutil.BaseTime.Add(8 * time.Second)
		lostAt := testutil.BaseTime.Add(4 * time.Second)
		beforeObserver := eventObserver(1, domain.ObserverOutbound, nil, &portalID, &startedAt, &transitEndsAt)
		afterObserver := eventObserver(1, domain.ObserverLost, nil, nil, nil, nil)
		afterObserver.UpdatedAt = lostAt
		portal := testutil.NewPortalBuilder().Build()
		portal.ID = portalID
		portal.DestinationPlaneID = 2

		before := domain.SimulationState{Portals: []domain.Portal{portal}, Observers: []domain.Observer{beforeObserver}}
		after := domain.SimulationState{Portals: []domain.Portal{portal}, Observers: []domain.Observer{afterObserver}}
		events := transitionEvents(t, before, after, testutil.BaseTime, lostAt)
		require.Equal(t, []domain.EventType{domain.EventObserverLost}, eventTypes(events))
		require.Equal(t, lostAt, events[0].CreatedAt)
		require.NotNil(t, events[0].PlaneID)
		require.Equal(t, int64(2), *events[0].PlaneID)
	})
}

func TestEventsForTransition_ExtractionSyncBeforeAutomaticReturn(t *testing.T) {
	portal := tickExtractionPortal(2, 1, 1, testutil.BaseTime)
	afterPortal := portal
	syncAt := testutil.BaseTime.Add(config.Default().ExtractionSync)
	afterPortal.ExtractionSynchronizedAt = &syncAt
	observerID, planeID := int64(1), int64(1)
	waitStarted := testutil.BaseTime.Add(-time.Second)
	beforeObserver := eventObserver(observerID, domain.ObserverWaitingReturn, &planeID, nil, &waitStarted, nil)
	returnEndsAt := syncAt.Add(5 * time.Second)
	afterObserver := eventObserver(observerID, domain.ObserverReturning, &planeID, &afterPortal.ID, &syncAt, &returnEndsAt)

	before := domain.SimulationState{Portals: []domain.Portal{portal}, Observers: []domain.Observer{beforeObserver}}
	after := domain.SimulationState{Portals: []domain.Portal{afterPortal}, Observers: []domain.Observer{afterObserver}}
	events := transitionEvents(t, before, after, testutil.BaseTime, syncAt)
	require.Equal(t, []domain.EventType{domain.EventExtractionSynchronized, domain.EventObserverReturnStarted}, eventTypes(events))
	require.Equal(t, []time.Time{syncAt, syncAt}, eventTimes(events))
}

func TestEventsForTransition_RiskBandChangeOnly(t *testing.T) {
	portal := riskTransitionPortal()
	before := stateWithPortals(portal)
	after := stateWithPortals(portal)
	now := testutil.BaseTime.Add(30 * time.Second)

	events := transitionEvents(t, before, after, testutil.BaseTime, now)
	require.Len(t, events, 1)
	require.Equal(t, domain.EventRiskLevelChanged, events[0].EventType)
	assert.JSONEq(t, `{"previous":"LOW","current":"MEDIUM"}`, events[0].PayloadJSON)

	unchanged := transitionEvents(t, before, after, testutil.BaseTime, testutil.BaseTime.Add(time.Second))
	require.Empty(t, unchanged)
}

func TestEventsForTransition_MultiBandJumpIsSingleEvent(t *testing.T) {
	portal := riskTransitionPortal()
	events := transitionEvents(t, stateWithPortals(portal), stateWithPortals(portal), testutil.BaseTime, testutil.BaseTime.Add(50*time.Second))
	require.Len(t, events, 1)
	require.Equal(t, domain.EventRiskLevelChanged, events[0].EventType)
	assert.JSONEq(t, `{"previous":"LOW","current":"CRITICAL"}`, events[0].PayloadJSON)
}

func TestEventsForTransition_OpenAndTerminalDoNotEmitRiskChange(t *testing.T) {
	portal := riskTransitionPortal()
	opened := transitionEvents(t, domain.SimulationState{}, stateWithPortals(portal), testutil.BaseTime, testutil.BaseTime.Add(50*time.Second))
	require.Equal(t, []domain.EventType{domain.EventPortalOpened}, eventTypes(opened))

	terminal := portal
	closedAt := testutil.BaseTime.Add(50 * time.Second)
	makeTerminalPortal(&terminal, domain.PortalStatusClosed, domain.TerminationNaturalClose, closedAt)
	closed := transitionEvents(t, stateWithPortals(portal), stateWithPortals(terminal), testutil.BaseTime, closedAt)
	require.Equal(t, []domain.EventType{domain.EventPortalClosed}, eventTypes(closed))
}

func transitionEvents(t *testing.T, before, after domain.SimulationState, previousAt, now time.Time) []domain.EventDraft {
	t.Helper()
	events, err := domain.EventsForStateTransition(before, after, previousAt, now, config.Default())
	require.NoError(t, err)
	for _, event := range events {
		require.NoError(t, event.Validate())
	}
	return events
}

func stateWithPortals(portals ...domain.Portal) domain.SimulationState {
	return domain.SimulationState{Portals: portals}
}

func stateWithObservers(observers ...domain.Observer) domain.SimulationState {
	return domain.SimulationState{Observers: observers}
}

func eventObserver(id int64, status domain.ObserverStatus, planeID, portalID *int64, startedAt, endsAt *time.Time) domain.Observer {
	return domain.Observer{
		ID: id, Status: status, CurrentPlaneID: planeID, ActivePortalID: portalID,
		PhaseStartedAt: startedAt, PhaseEndsAt: endsAt,
		CreatedAt: testutil.BaseTime, UpdatedAt: testutil.BaseTime,
	}
}

func riskTransitionPortal() domain.Portal {
	portal := testutil.NewPortalBuilder().TTL(60 * time.Second).Energy(100).Decay(1).Build()
	return portal
}

func eventTypes(events []domain.EventDraft) []domain.EventType {
	result := make([]domain.EventType, len(events))
	for i := range events {
		result[i] = events[i].EventType
	}
	return result
}

func eventTimes(events []domain.EventDraft) []time.Time {
	result := make([]time.Time, len(events))
	for i := range events {
		result[i] = events[i].CreatedAt
	}
	return result
}
