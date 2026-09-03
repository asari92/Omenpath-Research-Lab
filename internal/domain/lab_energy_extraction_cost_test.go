package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func TestLabState_ExtractionCostIsThirty(t *testing.T) {
	require.Equal(t, 30, config.Default().ExtractionCost)
}

func TestLabState_SpendExtractionCostAllowsExactThirty(t *testing.T) {
	cfg := config.Default()
	lab := labAt(cfg.ExtractionCost, testutil.BaseTime)

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime, cfg.ExtractionCost, cfg))
}

func TestLabState_SpendExtractionCostRejectsTwentyNine(t *testing.T) {
	cfg := config.Default()
	lab := labAt(cfg.ExtractionCost-1, testutil.BaseTime)

	err := lab.SpendEnergy(testutil.BaseTime, cfg.ExtractionCost, cfg)
	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
}

func TestLabState_SpendExtractionCostRebasesToZero(t *testing.T) {
	cfg := config.Default()
	lab := labAt(cfg.ExtractionCost, testutil.BaseTime)

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime, cfg.ExtractionCost, cfg))
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, testutil.BaseTime, lab.EnergyBaseAt)
}

func TestLabState_SpendExtractionCostUsesRegeneratedEnergy(t *testing.T) {
	cfg := config.Default()
	lab := labAt(cfg.ExtractionCost-1, testutil.BaseTime)
	now := testutil.BaseTime.Add(time.Second)

	require.NoError(t, lab.SpendEnergy(now, cfg.ExtractionCost, cfg))
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestLabState_SpendExtractionCostDoesNotCreatePortal(t *testing.T) {
	cfg := config.Default()
	lab := labAt(cfg.ExtractionCost, testutil.BaseTime)
	var portals []domain.Portal

	require.NoError(t, lab.SpendEnergy(testutil.BaseTime, cfg.ExtractionCost, cfg))
	require.Empty(t, portals)
}
