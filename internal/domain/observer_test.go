package domain_test

// Stage 3, checkpoint A (05_STAGE_03_OBSERVER_LIFECYCLE_TDD.md):
// Observer construction, the configured roster size, and canonical
// AVAILABLE/LOST state representation.

import (
	"testing"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestNewObserver_StartsAvailableInLaboratory(t *testing.T) {
	observer := domain.NewObserver(7, testutil.BaseTime)

	require.Equal(t, int64(7), observer.ID)
	require.Equal(t, domain.ObserverAvailable, observer.Status)
	require.Equal(t, testutil.BaseTime, observer.CreatedAt)
	require.Equal(t, testutil.BaseTime, observer.UpdatedAt)
}

func TestNewObserverRoster_CreatesConfiguredCount(t *testing.T) {
	roster := domain.NewObserverRoster(3, testutil.BaseTime)

	require.Len(t, roster, 3)
	require.Equal(t, []int64{1, 2, 3}, []int64{roster[0].ID, roster[1].ID, roster[2].ID})
	for _, observer := range roster {
		require.Equal(t, domain.ObserverAvailable, observer.Status)
	}
}

func TestNewObserverRoster_DefaultConfigCreatesTen(t *testing.T) {
	roster := domain.NewObserverRoster(config.Default().ObserverCount, testutil.BaseTime)

	require.Len(t, roster, 10)
}

func TestObserver_AvailableCanonicalFields(t *testing.T) {
	observer := domain.NewObserver(1, testutil.BaseTime)

	require.Equal(t, domain.ObserverAvailable, observer.Status)
	require.Nil(t, observer.CurrentPlaneID)
	require.Nil(t, observer.ActivePortalID)
	require.Nil(t, observer.PhaseStartedAt)
	require.Nil(t, observer.PhaseEndsAt)
	require.False(t, observer.IsTerminal())
}

func TestObserver_LostIsTerminal(t *testing.T) {
	observer := domain.Observer{Status: domain.ObserverLost}

	require.True(t, observer.IsTerminal())
}
