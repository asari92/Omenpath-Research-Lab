package random

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRealRandom_ImplementsRandom(t *testing.T) {
	var _ Random = NewRealRandom()
}

func TestRealRandom_IntInclusiveStaysWithinBounds(t *testing.T) {
	r := NewRealRandom()
	for i := 0; i < 1000; i++ {
		v := r.IntInclusive(3, 9)
		require.GreaterOrEqual(t, v, 3)
		require.LessOrEqual(t, v, 9)
	}
}

func TestRealRandom_IntInclusiveDegenerateRange(t *testing.T) {
	r := NewRealRandom()
	for i := 0; i < 10; i++ {
		require.Equal(t, 5, r.IntInclusive(5, 5))
	}
}

func TestRealRandom_FloatRangeStaysWithinBounds(t *testing.T) {
	r := NewRealRandom()
	for i := 0; i < 1000; i++ {
		v := r.FloatRange(0.1, 0.9)
		require.GreaterOrEqual(t, v, 0.1)
		require.LessOrEqual(t, v, 0.9)
	}
}

func TestRealRandom_PanicsOnEmptyRange(t *testing.T) {
	r := NewRealRandom()
	require.Panics(t, func() { r.IntInclusive(5, 4) })
	require.Panics(t, func() { r.FloatRange(0.9, 0.1) })
}

// Guard for the -race quality gate: RealRandom must be concurrency-safe.
func TestRealRandom_ConcurrentUse(t *testing.T) {
	r := NewRealRandom()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.IntInclusive(0, 100)
			_ = r.FloatRange(0, 1)
		}()
	}
	wg.Wait()
}

func TestRealRandom_MarshalRestoreReplaysExactPCGState(t *testing.T) {
	r := NewRealRandom()
	checkpointable, ok := any(r).(interface {
		MarshalBinary() ([]byte, error)
		UnmarshalBinary([]byte) error
	})
	if !ok {
		t.Fatal("RealRandom does not expose a checkpoint boundary")
	}
	state, err := checkpointable.MarshalBinary()
	require.NoError(t, err)

	wantInt := r.IntInclusive(-1000, 1000)
	wantFloat := r.FloatRange(-5.5, 9.25)
	wantNext := r.IntInclusive(1, 1_000_000)
	require.NoError(t, checkpointable.UnmarshalBinary(state))

	require.Equal(t, wantInt, r.IntInclusive(-1000, 1000))
	require.Equal(t, wantFloat, r.FloatRange(-5.5, 9.25))
	require.Equal(t, wantNext, r.IntInclusive(1, 1_000_000))
}
