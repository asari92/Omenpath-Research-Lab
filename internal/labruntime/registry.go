// Package labruntime owns one manager, random stream and realtime hub per lab.
package labruntime

import (
	"context"
	"errors"
	"sync"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/random"
	"omenpath-lab/internal/realtime"
)

var ErrClosed = errors.New("laboratory runtime registry closed")

type Runtime struct {
	Manager *engine.LabManager
	Hub     *realtime.Hub
}
type entry struct {
	runtime *Runtime
	refs    int
	busy    chan struct{}
}
type Registry struct {
	mu         sync.Mutex
	entries    map[persistence.LabID]*entry
	store      *persistence.Store
	cfg        config.Config
	clock      clock.Clock
	closed     bool
	operations sync.WaitGroup
	closeOnce  sync.Once
}
type Lease struct {
	Runtime *Runtime
	release func()
	once    sync.Once
}

func (l *Lease) Release() {
	if l != nil {
		l.once.Do(func() {
			if l.release != nil {
				l.release()
			}
		})
	}
}

func New(store *persistence.Store, cfg config.Config, c clock.Clock) (*Registry, error) {
	if store == nil || c == nil {
		return nil, errors.New("invalid runtime dependencies")
	}
	return &Registry{entries: make(map[persistence.LabID]*entry), store: store, cfg: cfg, clock: c}, nil
}

// Acquire coalesces concurrent loads. The global mutex protects admission only;
// database work is performed outside it while waiters observe the entry gate.
func (r *Registry) Acquire(ctx context.Context, id persistence.LabID) (*Lease, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		r.mu.Lock()
		if r.closed {
			r.mu.Unlock()
			return nil, ErrClosed
		}
		e := r.entries[id]
		if e != nil && e.busy != nil {
			done := e.busy
			r.mu.Unlock()
			select {
			case <-done:
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		if e != nil && e.runtime != nil {
			lease := r.leaseLocked(e)
			r.mu.Unlock()
			return lease, nil
		}
		e = &entry{busy: make(chan struct{})}
		r.entries[id] = e
		r.operations.Add(1)
		r.mu.Unlock()
		repo, err := r.store.ForLab(id)
		var runtime *Runtime
		if err == nil {
			var manager *engine.LabManager
			manager, err = engine.NewLabManager(ctx, r.cfg, r.clock, random.NewRealRandom(), repo)
			if err == nil {
				var hub *realtime.Hub
				hub, err = realtime.NewHub(manager, r.cfg)
				if err == nil {
					runtime = &Runtime{Manager: manager, Hub: hub}
				}
			}
		}
		r.mu.Lock()
		e.runtime = runtime
		close(e.busy)
		e.busy = nil
		if err != nil {
			delete(r.entries, id)
			r.mu.Unlock()
			r.operations.Done()
			return nil, err
		}
		if r.closed {
			r.mu.Unlock()
			r.operations.Done()
			return nil, ErrClosed
		}
		lease := r.leaseLocked(e)
		r.mu.Unlock()
		r.operations.Done()
		return lease, nil
	}
}
func (r *Registry) leaseLocked(e *entry) *Lease {
	e.refs++
	return &Lease{Runtime: e.runtime, release: func() { r.mu.Lock(); e.refs--; r.mu.Unlock() }}
}

func (r *Registry) TickAll(ctx context.Context) error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return ErrClosed
	}
	leases := make([]*Lease, 0, len(r.entries))
	for _, e := range r.entries {
		if e.busy == nil && e.runtime != nil {
			leases = append(leases, r.leaseLocked(e))
		}
	}
	r.operations.Add(1)
	r.mu.Unlock()
	defer r.operations.Done()
	var result error
	for _, lease := range leases {
		result = errors.Join(result, lease.Runtime.Manager.Tick(ctx))
		lease.Release()
	}
	return result
}

// DeleteIfIdle serializes the conditional database deletion with acquisition.
// callback must recheck expiry; a stale candidate is not deletion authority.
// A successful callback may retain a renewed DB row; evicting its idle runtime
// remains safe, and the next Acquire reloads that persisted laboratory.
func (r *Registry) DeleteIfIdle(id persistence.LabID, remove func() error) (bool, error) {
	if err := id.Validate(); err != nil {
		return false, err
	}
	if remove == nil {
		return false, errors.New("nil laboratory delete callback")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return false, ErrClosed
	}
	e := r.entries[id]
	if e != nil && (e.refs > 0 || e.busy != nil) {
		r.mu.Unlock()
		return false, nil
	}
	if e == nil {
		e = &entry{}
		r.entries[id] = e
	}
	e.busy = make(chan struct{})
	r.operations.Add(1)
	r.mu.Unlock()
	err := remove()
	if err == nil && e.runtime != nil {
		e.runtime.Hub.Close()
	}
	r.mu.Lock()
	if err == nil {
		delete(r.entries, id)
	}
	close(e.busy)
	e.busy = nil
	r.mu.Unlock()
	r.operations.Done()
	return err == nil, err
}

// Close rejects new leases, awaits in-flight creation/ticks/deletion, and closes
// hubs (which drain WebSocket handlers and release their leases). Composition
// first drains HTTP and workers before closing the registry and database.
func (r *Registry) Close() {
	if r == nil {
		return
	}
	r.closeOnce.Do(func() {
		r.mu.Lock()
		r.closed = true
		r.mu.Unlock()
		r.operations.Wait()
		r.mu.Lock()
		runtimes := make([]*Runtime, 0, len(r.entries))
		for _, e := range r.entries {
			if e.runtime != nil {
				runtimes = append(runtimes, e.runtime)
			}
		}
		r.mu.Unlock()
		for _, runtime := range runtimes {
			runtime.Hub.Close()
		}
	})
}
