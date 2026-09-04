// Package random abstracts randomness for the whole codebase.
//
// Stage 1 rule: production code draws values only through Random.
// Domain tests use testutil.FakeRandom with preloaded deterministic values,
// so no real randomness leaks into domain test outcomes.
package random

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"
)

// Random is the single source of randomness for production code.
type Random interface {
	// IntInclusive returns a uniformly random int in [min, max].
	// It panics if min > max (empty range is a programming error).
	IntInclusive(min, max int) int

	// FloatRange returns a uniformly random float64 in [min, max).
	// It panics if min > max.
	FloatRange(min, max float64) float64
}

// Checkpointable is the transactional random boundary used by orchestration
// that must roll back draws when durable state cannot be committed. Pure
// domain consumers continue to depend only on Random.
type Checkpointable interface {
	Random
	MarshalBinary() ([]byte, error)
	UnmarshalBinary([]byte) error
}

// RealRandom is the production implementation.
// It is safe for concurrent use.
type RealRandom struct {
	mu     sync.Mutex
	source *rand.PCG
	rng    *rand.Rand
}

var _ Random = (*RealRandom)(nil)
var _ Checkpointable = (*RealRandom)(nil)

// NewRealRandom seeds the generator from crypto/rand.
func NewRealRandom() *RealRandom {
	a, b := seed()
	source := rand.NewPCG(a, b)
	return &RealRandom{source: source, rng: rand.New(source)}
}

// IntInclusive implements Random.
func (r *RealRandom) IntInclusive(min, max int) int {
	if min > max {
		panic(fmt.Sprintf("random: empty int range min=%d max=%d", min, max))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.IntN(max-min+1) + min
}

// FloatRange implements Random.
func (r *RealRandom) FloatRange(min, max float64) float64 {
	if min > max {
		panic(fmt.Sprintf("random: empty float range min=%f max=%f", min, max))
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rng.Float64()*(max-min) + min
}

// MarshalBinary returns an exact snapshot of the underlying PCG state.
func (r *RealRandom) MarshalBinary() ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.source == nil {
		return nil, fmt.Errorf("random: uninitialized PCG source")
	}
	return r.source.MarshalBinary()
}

// UnmarshalBinary restores an exact snapshot of the underlying PCG state.
func (r *RealRandom) UnmarshalBinary(state []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.source == nil {
		return fmt.Errorf("random: uninitialized PCG source")
	}
	if err := r.source.UnmarshalBinary(state); err != nil {
		return fmt.Errorf("random: restore PCG state: %w", err)
	}
	return nil
}

func seed() (uint64, uint64) {
	var buf [16]byte
	if _, err := cryptorand.Read(buf[:]); err == nil {
		return binary.LittleEndian.Uint64(buf[:8]), binary.LittleEndian.Uint64(buf[8:])
	}
	// Deterministic fallback should crypto/rand ever fail; keeps the
	// constructor total. Time-based mixing is enough for game balance.
	now := uint64(time.Now().UnixNano())
	return now, now ^ 0x9E3779B97F4A7C15
}
