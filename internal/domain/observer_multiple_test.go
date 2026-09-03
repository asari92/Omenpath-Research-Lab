package domain_test

// Stage 3, checkpoint H characterization: lifecycle primitives intentionally
// own no global "one Observer per Plane" rule (Final Spec §15). Portal busy
// admission is separately deferred to Stage 4.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestObservers_MultipleObserversMayExploreSamePlaneConcurrently(t *testing.T) {
	cfg := config.Default()
	transitEnd := testutil.BaseTime.Add(10 * time.Second)
	first := outboundObserver(testutil.BaseTime, transitEnd)
	second := outboundObserver(testutil.BaseTime, transitEnd)
	second.ID = 2
	plane := observerTestPlane()
	portal := observerTestPortal()

	// Both already-authorized transits may reach the same Plane. Enforcing one
	// transit per Portal is command admission owned by Stage 4, not this core.
	require.NoError(t, domain.ResolveObserverLifecycle(&first, &plane, &portal, transitEnd, cfg))
	require.NoError(t, domain.ResolveObserverLifecycle(&second, &plane, &portal, transitEnd, cfg))
	require.Equal(t, domain.ObserverExploring, first.Status)
	require.Equal(t, domain.ObserverExploring, second.Status)
	require.Equal(t, plane.ID, *first.CurrentPlaneID)
	require.Equal(t, plane.ID, *second.CurrentPlaneID)

	researchEnd := transitEnd.Add(cfg.ResearchDuration)
	require.NoError(t, domain.ResolveObserverLifecycle(&first, &plane, &portal, researchEnd, cfg))
	require.NoError(t, domain.ResolveObserverLifecycle(&second, &plane, &portal, researchEnd, cfg))
	require.Equal(t, domain.ObserverWaitingReturn, first.Status)
	require.Equal(t, domain.ObserverWaitingReturn, second.Status)
	require.False(t, plane.Explored)

	firstReturnStart := researchEnd
	secondReturnStart := researchEnd.Add(time.Second)
	require.NoError(t, first.StartReturning(firstReturnStart, portal.ID, testutil.NewFakeRandom().QueueInt(10), cfg))
	require.NoError(t, second.StartReturning(secondReturnStart, portal.ID, testutil.NewFakeRandom().QueueInt(10), cfg))

	firstReturnEnd := firstReturnStart.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&first, &plane, &portal, firstReturnEnd, cfg))
	require.True(t, plane.Explored)
	require.NotNil(t, plane.ExploredAt)
	require.Equal(t, firstReturnEnd, *plane.ExploredAt)

	secondReturnEnd := secondReturnStart.Add(10 * time.Second)
	require.NoError(t, domain.ResolveObserverLifecycle(&second, &plane, &portal, secondReturnEnd, cfg))
	require.Equal(t, domain.ObserverAvailable, second.Status)
	require.Equal(t, firstReturnEnd, *plane.ExploredAt, "later return must preserve first exploration time")
}
