// Package testutil contains deterministic test infrastructure:
// FakeClock, FakeRandom and fixture builders.
//
// These are test helpers only — production code must never import them.
package testutil

import (
	"sync"
	"time"

	"omenpath-lab/internal/clock"
)

// BaseTime is the canonical deterministic base instant for domain tests
// (Stage 2 plan §24: fixed UTC date, no local-timezone dependencies).
var BaseTime = time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

// FakeClock is a deterministic Clock for tests.
// It is safe for concurrent use (required by `go test -race`).
type FakeClock struct {
	mu      sync.Mutex
	current time.Time
}

var _ clock.Clock = (*FakeClock)(nil)

// NewFakeClock returns a FakeClock set to at.
func NewFakeClock(at time.Time) *FakeClock {
	return &FakeClock{current: at}
}

// Now implements clock.Clock.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

// Advance moves the clock forward by d and returns the new instant.
// Successive calls accumulate; no sleeping is ever involved.
func (c *FakeClock) Advance(d time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = c.current.Add(d)
	return c.current
}
