package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRealClock_ImplementsClock(t *testing.T) {
	var _ Clock = RealClock{} // compile-time contract check
}

func TestRealClock_ReturnsCurrentTime(t *testing.T) {
	c := RealClock{}

	before := time.Now()
	got := c.Now()
	after := time.Now()

	require.False(t, got.Before(before), "Now() must not precede the call instant")
	require.False(t, got.After(after), "Now() must not exceed the instant after the call")
}
