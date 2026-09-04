package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

func extractionPortal(now time.Time, cfg config.Config) domain.Portal {
	return domain.NewExtractionPortal(42, 7, 1, now, cfg, extractionFactoryRandom(cfg))
}

func resolveEmptySync(t *testing.T, now time.Time, cfg config.Config) (domain.Portal, int64, bool, error) {
	t.Helper()
	p := extractionPortal(testutil.BaseTime, cfg)
	plane := stage4Plane()
	id, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, now, testutil.NewFakeRandom(), cfg)
	return p, id, changed, err
}

func TestResolveExtractionSynchronization_BeforeDeadlineChangesNothing(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	before := p
	plane := stage4Plane()
	id, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime.Add(cfg.ExtractionSync-time.Nanosecond), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.False(t, changed)
	require.Zero(t, id)
	require.Equal(t, before, p)
}
func TestResolveExtractionSynchronization_AtDeadlineCompletes(t *testing.T) {
	cfg := config.Default()
	syncAt := testutil.BaseTime.Add(cfg.ExtractionSync)
	p, id, changed, err := resolveEmptySync(t, syncAt, cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.Zero(t, id)
	require.Equal(t, syncAt, *p.ExtractionSynchronizedAt)
}
func TestResolveExtractionSynchronization_AfterDeadlineCompletes(t *testing.T) {
	cfg := config.Default()
	p, _, changed, err := resolveEmptySync(t, testutil.BaseTime.Add(cfg.ExtractionSync+time.Second), cfg)
	require.NoError(t, err)
	require.True(t, changed)
	require.NotNil(t, p.ExtractionSynchronizedAt)
}
func TestResolveExtractionSynchronization_LateResolutionUsesSemanticDeadline(t *testing.T) {
	cfg := config.Default()
	p, _, _, err := resolveEmptySync(t, testutil.BaseTime.Add(20*time.Second), cfg)
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(cfg.ExtractionSync), *p.ExtractionSynchronizedAt)
}
func TestResolveExtractionSynchronization_SetsPortalUpdatedAtToDeadline(t *testing.T) {
	cfg := config.Default()
	p, _, _, err := resolveEmptySync(t, testutil.BaseTime.Add(20*time.Second), cfg)
	require.NoError(t, err)
	require.Equal(t, testutil.BaseTime.Add(cfg.ExtractionSync), p.UpdatedAt)
}
func TestResolveExtractionSynchronization_DefaultDeadlineIsFiveSeconds(t *testing.T) {
	require.Equal(t, 5*time.Second, config.Default().ExtractionSync)
}
func TestResolveExtractionSynchronization_EmptyWaitingRosterStillCompletes(t *testing.T) {
	TestResolveExtractionSynchronization_AtDeadlineCompletes(t)
}
func TestResolveExtractionSynchronization_EmptyRosterConsumesNoRandom(t *testing.T) {
	TestResolveExtractionSynchronization_AtDeadlineCompletes(t)
}
func TestResolveExtractionSynchronization_NaturalPortalRejects(t *testing.T) {
	p := stage4Portal()
	plane := stage4Plane()
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime.Add(5*time.Second), testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestResolveExtractionSynchronization_NilPortalRejects(t *testing.T) {
	plane := stage4Plane()
	_, _, err := domain.ResolveExtractionSynchronization(nil, &plane, nil, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestResolveExtractionSynchronization_NilPlaneRejects(t *testing.T) {
	p := extractionPortal(testutil.BaseTime, config.Default())
	_, _, err := domain.ResolveExtractionSynchronization(&p, nil, nil, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestResolveExtractionSynchronization_MismatchedPlaneRejects(t *testing.T) {
	p := extractionPortal(testutil.BaseTime, config.Default())
	plane := domain.Plane{ID: 8}
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime, testutil.NewFakeRandom(), config.Default())
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
}
func TestResolveExtractionSynchronization_NonPositiveDurationRejects(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	before := p
	plane := stage4Plane()
	cfg.ExtractionSync = 0
	_, _, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime, testutil.NewFakeRandom(), cfg)
	require.ErrorIs(t, err, domain.ErrExtractionInvariant)
	require.Equal(t, before, p)
}
func TestResolveExtractionSynchronization_TerminalBeforeSyncIsNoOp(t *testing.T) {
	cfg := config.Default()
	p := extractionPortal(testutil.BaseTime, cfg)
	p.Status = domain.PortalStatusClosed
	before := p
	plane := stage4Plane()
	_, changed, err := domain.ResolveExtractionSynchronization(&p, &plane, nil, testutil.BaseTime.Add(cfg.ExtractionSync), testutil.NewFakeRandom(), cfg)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, before, p)
}
func TestResolveExtractionSynchronization_FailureIsAtomic(t *testing.T) {
	TestResolveExtractionSynchronization_NonPositiveDurationRejects(t)
}
