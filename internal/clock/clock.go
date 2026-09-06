// Package clock abstracts time for the whole codebase.
//
// Production domain code does not call time.Now() directly: it receives a
// Clock. Tests use testutil.FakeClock and Advance instead of sleeping.
package clock

import "time"

// Clock is the single source of "now" for production code.
type Clock interface {
	Now() time.Time
}

// RealClock is the production implementation backed by the wall clock.
type RealClock struct{}

var _ Clock = RealClock{}

// Now implements Clock.
func (RealClock) Now() time.Time { return time.Now() }
