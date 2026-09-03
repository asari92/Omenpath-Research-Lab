package testutil

import (
	"fmt"

	"omenpath-lab/internal/random"
)

// FakeRandom is a deterministic Random for tests.
//
// Queued values are returned as-is, in order, regardless of the requested
// range: fixtures must be explicit and honest (no hidden clamping).
// Exhausting the queue panics — a test must fail loudly rather than
// silently draw something unexpected.
type FakeRandom struct {
	ints   []int
	floats []float64
}

var _ random.Random = (*FakeRandom)(nil)

// NewFakeRandom returns an empty FakeRandom.
func NewFakeRandom() *FakeRandom {
	return &FakeRandom{}
}

// QueueInt appends preloaded int results.
func (f *FakeRandom) QueueInt(values ...int) *FakeRandom {
	f.ints = append(f.ints, values...)
	return f
}

// QueueFloat appends preloaded float64 results.
func (f *FakeRandom) QueueFloat(values ...float64) *FakeRandom {
	f.floats = append(f.floats, values...)
	return f
}

// IntInclusive implements random.Random.
func (f *FakeRandom) IntInclusive(min, max int) int {
	if len(f.ints) == 0 {
		panic("FakeRandom: int queue exhausted")
	}
	v := f.ints[0]
	f.ints = f.ints[1:]
	return v
}

// FloatRange implements random.Random.
func (f *FakeRandom) FloatRange(min, max float64) float64 {
	if len(f.floats) == 0 {
		panic(fmt.Sprintf("FakeRandom: float queue exhausted"))
	}
	v := f.floats[0]
	f.floats = f.floats[1:]
	return v
}
