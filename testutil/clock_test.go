package testutil

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/clock"
)

func TestFakeClock_ImplementsClock(t *testing.T) {
	var _ clock.Clock = NewFakeClock(BaseTime)
}

func TestFakeClock_ReturnsBaseTimeStably(t *testing.T) {
	c := NewFakeClock(BaseTime)
	require.Equal(t, BaseTime, c.Now())
	require.Equal(t, BaseTime, c.Now()) // repeated calls must not drift
}

func TestFakeClock_AdvanceAccumulates(t *testing.T) {
	c := NewFakeClock(BaseTime)

	got := c.Advance(10 * time.Second)
	require.Equal(t, BaseTime.Add(10*time.Second), got)

	got = c.Advance(5 * time.Second)
	require.Equal(t, BaseTime.Add(15*time.Second), got)
	require.Equal(t, BaseTime.Add(15*time.Second), c.Now())
}

func TestFakeClock_AdvanceSubSecondPrecision(t *testing.T) {
	c := NewFakeClock(BaseTime)
	c.Advance(1500 * time.Millisecond)
	require.Equal(t, BaseTime.Add(1500*time.Millisecond), c.Now())
}

// Guard for the -race quality gate: concurrent Advance/Now must be safe.
func TestFakeClock_ConcurrentAdvanceAndNow(t *testing.T) {
	c := NewFakeClock(BaseTime)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); c.Advance(time.Second) }()
		go func() { defer wg.Done(); _ = c.Now() }()
	}
	wg.Wait()
	require.Equal(t, BaseTime.Add(8*time.Second), c.Now())
}
