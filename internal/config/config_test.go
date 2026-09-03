package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDefault_MatchesFinalSpecBalance is the executable part of Stage 0:
// it pins Config.Default() to the balance table of Final Spec §37.
// If this test fails, either the spec or the config changed — never tune
// it silently.
func TestDefault_MatchesFinalSpecBalance(t *testing.T) {
	cfg := Default()

	require.Equal(t, 7, cfg.MaxActivePortals)
	require.Equal(t, 0*time.Second, cfg.SpawnDelayMin)
	require.Equal(t, 20*time.Second, cfg.SpawnDelayMax)

	require.Equal(t, 10*time.Second, cfg.NaturalTTLMin)
	require.Equal(t, 300*time.Second, cfg.NaturalTTLMax)

	require.Equal(t, 10.0, cfg.PortalEnergyMin)
	require.Equal(t, 100.0, cfg.PortalEnergyMax)
	require.Equal(t, 0.1, cfg.PortalDecayMin)
	require.Equal(t, 1.0, cfg.PortalDecayMax)

	require.Equal(t, 0.35, cfg.UnstableProbability)
	require.Equal(t, 5*time.Second, cfg.InstabilityMinLifetime)
	require.Equal(t, 1*time.Second, cfg.InstabilityCloseMargin)

	require.Equal(t, 10, cfg.CreatureMax)
	require.Equal(t, 2*time.Second, cfg.CreatureTransit)
	require.Equal(t, 2*time.Second, cfg.CreatureClearanceMargin)

	require.Equal(t, 10, cfg.ObserverCount)
	require.Equal(t, 5*time.Second, cfg.ObserverTransitMin)
	require.Equal(t, 15*time.Second, cfg.ObserverTransitMax)
	require.Equal(t, 20*time.Second, cfg.ResearchDuration)

	require.Equal(t, 100, cfg.LabEnergyMax)
	require.Equal(t, 1, cfg.LabRegenPerSec)
	require.Equal(t, 5, cfg.CloseCost)
	require.Equal(t, 20, cfg.StabilizeCost)
	require.Equal(t, 15.0, cfg.StabilizeBoost)
	require.Equal(t, 85.0, cfg.StabilizeMaxStartEnergy)
	require.Equal(t, 30, cfg.ExtractionCost)
	require.Equal(t, 60.0, cfg.ExtractionEnergyMin)
	require.Equal(t, 100.0, cfg.ExtractionEnergyMax)
	require.Equal(t, 30*time.Second, cfg.ExtractionTTLMin)
	require.Equal(t, 60*time.Second, cfg.ExtractionTTLMax)
	require.Equal(t, 5*time.Second, cfg.ExtractionSync)

	require.Equal(t, 20*time.Second, cfg.EmergencyDuration)

	require.Equal(t, 45*time.Second, cfg.RiskSafeHorizon)
	require.Equal(t, 20.0, cfg.RiskInstabilityPenalty)
}
