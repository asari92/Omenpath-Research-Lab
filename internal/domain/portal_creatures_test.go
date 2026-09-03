package domain_test

// Stage 2, substage 2.7 (04_STAGE_02_PORTAL_CORE_TDD.md): creatures.
// Final Spec §12: 2 sec per creature, 2 sec clearance margin, derived count.

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/testutil"
)

// CREATURE-003/004/005 (Final Spec §12): max creatures for a TTL.
func TestMaxCreaturesForTTL(t *testing.T) {
	cases := []struct {
		name string
		ttl  time.Duration
		want int
	}{
		{"TTL 10s allows four", 10 * time.Second, 4},    // floor((10-2)/2)
		{"TTL 20s allows nine", 20 * time.Second, 9},    // floor((20-2)/2)
		{"TTL 30s capped at ten", 30 * time.Second, 10}, // 14 → cap
		{"TTL 22s hits the cap exactly", 22 * time.Second, 10},
		{"TTL 4s allows one", 4 * time.Second, 1},
		{"defensive: TTL 1s allows zero", 1 * time.Second, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, domain.MaxCreaturesForTTL(tc.ttl, config.Default()))
		})
	}
}

// CREATURE-006 (Final Spec §12): current count is derived — one creature
// leaves every 2 seconds, clamped at zero.
func TestPortal_CreaturesInsideDecreasesEveryTwoSeconds(t *testing.T) {
	p := testutil.NewPortalBuilder().Creatures(4).Build()

	cases := []struct {
		elapsed time.Duration
		want    int
	}{
		{0, 4},
		{1 * time.Second, 4},
		{2 * time.Second, 3},
		{3 * time.Second, 3},
		{4 * time.Second, 2},
		{6 * time.Second, 1},
		{8 * time.Second, 0},
		{20 * time.Second, 0},
	}
	for _, tc := range cases {
		t.Run(tc.elapsed.String(), func(t *testing.T) {
			now := testutil.BaseTime.Add(tc.elapsed)
			require.Equal(t, tc.want, p.CreaturesInside(now, config.Default()))
		})
	}
}
