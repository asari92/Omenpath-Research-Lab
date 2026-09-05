package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestPortalActionAvailability_ReturnsCommandCauseWithoutMutation(t *testing.T) {
	now := testutil.BaseTime
	cases := []struct {
		name   string
		action domain.PortalAction
		mutate func(*domain.SimulationState)
		cause  error
	}{
		{"stable cannot stabilize", domain.PortalActionStabilize, func(*domain.SimulationState) {}, domain.ErrPortalAlreadyStable},
		{"overcharge", domain.PortalActionStabilize, func(s *domain.SimulationState) {
			s.Portals[0] = testutil.NewPortalBuilder().Energy(90).Unstable(now.Add(30 * time.Second)).Build()
		}, domain.ErrPortalOverchargeRisk},
		{"insufficient stabilize energy", domain.PortalActionStabilize, func(s *domain.SimulationState) {
			s.Portals[0] = testutil.NewPortalBuilder().Energy(50).Unstable(now.Add(30 * time.Second)).Build()
			s.Lab.EnergyBase = 0
		}, domain.ErrInsufficientLabEnergy},
		{"terminal", domain.PortalActionClose, func(s *domain.SimulationState) {
			closed := now
			s.Portals[0].Status = domain.PortalStatusClosed
			s.Portals[0].TerminationReason = domain.TerminationManualClose
			s.Portals[0].ClosedAt = &closed
			s.Portals[0].UpdatedAt = closed
		}, domain.ErrPortalNotOpen},
		{"critical send", domain.PortalActionSend, func(s *domain.SimulationState) {
			s.Portals[0] = testutil.NewPortalBuilder().Energy(10).TTL(10 * time.Second).Build()
		}, domain.ErrPortalCriticalRisk},
		{"send direction", domain.PortalActionSend, func(s *domain.SimulationState) {
			s.Portals[0].ObserverFlow = domain.PortalFlowInbound
		}, domain.ErrPortalDirectionConflict},
		{"creatures", domain.PortalActionSend, func(s *domain.SimulationState) {
			s.Portals[0].CreaturesInitial = 1
		}, domain.ErrPortalCreaturesPresent},
		{"busy", domain.PortalActionSend, func(s *domain.SimulationState) {
			start, end, portalID := now, now.Add(5*time.Second), int64(1)
			s.Observers[0].Status = domain.ObserverOutbound
			s.Observers[0].ActivePortalID = &portalID
			s.Observers[0].PhaseStartedAt = &start
			s.Observers[0].PhaseEndsAt = &end
		}, domain.ErrPortalBusy},
		{"no available", domain.PortalActionSend, func(s *domain.SimulationState) {
			for i := range s.Observers {
				s.Observers[i].Status = domain.ObserverLost
			}
		}, domain.ErrNoAvailableObserver},
		{"extraction synchronizing", domain.PortalActionRecall, func(s *domain.SimulationState) {
			s.Portals[0].Kind = domain.PortalKindExtraction
			s.Portals[0].ObserverFlow = domain.PortalFlowInbound
		}, domain.ErrExtractionSynchronizing},
		{"recall direction", domain.PortalActionRecall, func(s *domain.SimulationState) {
			s.Portals[0].ObserverFlow = domain.PortalFlowOutbound
		}, domain.ErrPortalDirectionConflict},
		{"no waiting", domain.PortalActionRecall, func(*domain.SimulationState) {}, domain.ErrNoWaitingObserver},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := availabilityState(now)
			tc.mutate(&state)
			before := state
			got, err := domain.PortalActionAvailability(state, 1, tc.action, now, config.Default())
			require.NoError(t, err)
			require.False(t, got.Available)
			require.ErrorIs(t, got.Cause, tc.cause)
			require.Equal(t, before, state)
		})
	}
}

func TestPortalActionAvailability_ConfirmationSensitiveCommandsAreAvailable(t *testing.T) {
	now := testutil.BaseTime
	state := availabilityState(now)
	state.Portals[0] = testutil.NewPortalBuilder().Energy(50).Unstable(now.Add(30 * time.Second)).Build()

	for _, action := range []domain.PortalAction{domain.PortalActionClose, domain.PortalActionSend} {
		got, err := domain.PortalActionAvailability(state, 1, action, now, config.Default())
		require.NoError(t, err)
		require.True(t, got.Available, action)
		require.NoError(t, got.Cause)
	}
}

func availabilityState(now time.Time) domain.SimulationState {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{ID: int64(i + 1), Name: "Plane"}
	}
	return domain.SimulationState{
		Lab:          domain.LabState{EnergyBase: 100, EnergyBaseAt: now},
		Portals:      []domain.Portal{testutil.NewPortalBuilder().Build()},
		Planes:       planes,
		Observers:    domain.NewObserverRoster(3, now.Add(-time.Hour)),
		NextPortalID: 2,
		NaturalSpawn: domain.NaturalSpawnState{Paused: true},
	}
}
