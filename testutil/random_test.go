package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/random"
)

func TestFakeRandom_ImplementsRandom(t *testing.T) {
	var _ random.Random = NewFakeRandom()
}

func TestFakeRandom_ReturnsQueuedValuesInOrder(t *testing.T) {
	r := NewFakeRandom().QueueInt(3, 7).QueueFloat(0.25, 0.75)

	require.Equal(t, 3, r.IntInclusive(0, 100))
	require.Equal(t, 7, r.IntInclusive(0, 100))
	require.InDelta(t, 0.25, r.FloatRange(0, 1), 1e-12)
	require.InDelta(t, 0.75, r.FloatRange(0, 1), 1e-12)
}

func TestFakeRandom_ValuesArePassedThroughAsIs(t *testing.T) {
	// Fixtures are explicit and honest: no clamping to the requested range.
	r := NewFakeRandom().QueueInt(50).QueueFloat(2.5)
	require.Equal(t, 50, r.IntInclusive(0, 10))
	require.InDelta(t, 2.5, r.FloatRange(0, 1), 1e-12)
}

func TestFakeRandom_PanicsWhenQueueExhausted(t *testing.T) {
	r := NewFakeRandom()
	require.Panics(t, func() { r.IntInclusive(1, 2) })
	require.Panics(t, func() { r.FloatRange(0, 1) })
}
