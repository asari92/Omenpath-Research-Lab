package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func unstablePortal(energy float64) domain.Portal {
	return testutil.NewPortalBuilder().
		Unstable(testutil.BaseTime.Add(30 * time.Second)).
		Energy(energy).
		Decay(0.5).
		TTL(time.Minute).
		Build()
}

func TestStabilizePortalWithLabEnergy_ChargesTwenty(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default()))
	require.Equal(t, 20, lab.EnergyBase)
}

func TestStabilizePortalWithLabEnergy_AllowsExactTwenty(t *testing.T) {
	lab := labAt(20, testutil.BaseTime)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default()))
	require.Zero(t, lab.EnergyBase)
}

func TestStabilizePortalWithLabEnergy_UsesRegeneratedEnergy(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	portal := unstablePortal(50)
	now := testutil.BaseTime.Add(time.Second)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, now, config.Default()))
	require.Zero(t, lab.EnergyBase)
	require.Equal(t, now, lab.EnergyBaseAt)
}

func TestStabilizePortalWithLabEnergy_RejectsNineteen(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	portal := unstablePortal(50)

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrInsufficientLabEnergy)
}

func TestStabilizePortalWithLabEnergy_InsufficientLeavesPortalUnchanged(t *testing.T) {
	lab := labAt(19, testutil.BaseTime)
	portal := unstablePortal(50)
	snapshot := portal

	_ = domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.Equal(t, snapshot, portal)
}

func TestStabilizePortalWithLabEnergy_SuccessPreservesPortalSemantics(t *testing.T) {
	lab := labAt(20, testutil.BaseTime)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default()))
	require.Equal(t, domain.PortalStable, portal.Stability)
	require.Nil(t, portal.InstabilityCollapseAt)
	require.InDelta(t, 65.0, portal.EnergyBase, 1e-9)
	require.Equal(t, testutil.BaseTime, portal.EnergyBaseAt)
	require.InDelta(t, 0.5, portal.EnergyDecayRate, 1e-9)
}

func TestStabilizePortalWithLabEnergy_StablePortalDoesNotDebit(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := stage4Portal()

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
	require.Equal(t, snapshot, lab)
}

func TestStabilizePortalWithLabEnergy_OverchargeRejectionDoesNotDebit(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := unstablePortal(86)

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalOverchargeRisk)
	require.Equal(t, snapshot, lab)
}

func TestStabilizePortalWithLabEnergy_TerminalPortalDoesNotDebit(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	snapshot := lab
	portal := unstablePortal(50)
	portal.Status = domain.PortalStatusClosed

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
	require.Equal(t, snapshot, lab)
}

func TestStabilizePortalWithLabEnergy_DebitsExactlyOnce(t *testing.T) {
	lab := labAt(40, testutil.BaseTime)
	portal := unstablePortal(50)

	require.NoError(t, domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default()))
	require.Equal(t, 25, lab.CurrentEnergy(testutil.BaseTime.Add(5*time.Second), config.Default()))
}

func TestStabilizePortalWithLabEnergy_RejectsInvalidLabStateBeforeMutation(t *testing.T) {
	lab := labAt(-1, testutil.BaseTime)
	portal := unstablePortal(50)
	snapshot := portal

	err := domain.StabilizePortalWithLabEnergy(&lab, &portal, testutil.BaseTime, config.Default())
	require.ErrorIs(t, err, domain.ErrLabEnergyInvariant)
	require.Equal(t, snapshot, portal)
}
