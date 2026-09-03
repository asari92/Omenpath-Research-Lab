// Package clock abstracts time for the whole codebase.
//
// Stage 1 rule (03_STAGE_00_01_TDD_FOUNDATION.md): production domain code
// must never call time.Now() directly — it receives a Clock. Tests use
// testutil.FakeClock and Advance instead of sleeping.
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
