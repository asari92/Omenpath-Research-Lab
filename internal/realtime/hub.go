// Package realtime publishes authoritative laboratory snapshots to WebSocket
// clients. It deliberately reuses transport.StateSnapshot, the same public
// view model served by REST.
package realtime

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/transport"
)

const writeTimeout = 5 * time.Second

// StateSource is the read/update boundary exposed by engine.LabManager.
type StateSource interface {
	State(context.Context) (persistence.Snapshot, error)
	Updates() <-chan struct{}
}

type queuedSnapshot struct {
	sequence uint64
	view     transport.StateSnapshot
}

type client struct {
	queue        chan queuedSnapshot
	cancel       context.CancelFunc
	lastSequence uint64 // protected by Hub.mu
}

// Hub owns connected clients and the single bridge from manager update edges.
// Each client has a capacity-one queue, so publishing never waits for network
// IO and a newer authoritative snapshot replaces a stale pending one.
type Hub struct {
	source StateSource
	cfg    config.Config

	mu           sync.Mutex
	clients      map[*client]struct{}
	bridgeCancel context.CancelFunc
	closed       bool

	snapshotMu sync.Mutex
	sequence   uint64
}

func NewHub(source StateSource, cfg config.Config) (*Hub, error) {
	if stateSourceIsNil(source) {
		return nil, fmt.Errorf("new realtime hub: nil state source")
	}
	if source.Updates() == nil {
		return nil, fmt.Errorf("new realtime hub: nil updates channel")
	}
	if cfg.MaxActivePortals <= 0 || cfg.LabEnergyMax <= 0 {
		return nil, fmt.Errorf("new realtime hub: invalid config: %w", domain.ErrSimulationInvariant)
	}
	return &Hub{source: source, cfg: cfg, clients: make(map[*client]struct{})}, nil
}

func stateSourceIsNil(source StateSource) bool {
	if source == nil {
		return true
	}
	value := reflect.ValueOf(source)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (h *Hub) capture(ctx context.Context) (queuedSnapshot, error) {
	h.snapshotMu.Lock()
	defer h.snapshotMu.Unlock()

	snapshot, err := h.source.State(ctx)
	if err != nil {
		return queuedSnapshot{}, err
	}
	if snapshot.Simulation.LastTickAt == nil || snapshot.Simulation.LastTickAt.IsZero() {
		return queuedSnapshot{}, domain.ErrSimulationInvariant
	}
	view, err := transport.BuildStateSnapshot(snapshot, snapshot.Simulation.LastTickAt.UTC(), h.cfg)
	if err != nil {
		return queuedSnapshot{}, err
	}
	h.sequence++
	return queuedSnapshot{sequence: h.sequence, view: view}, nil
}

func (h *Hub) register(ctx context.Context, candidate *client) error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return context.Canceled
	}
	h.clients[candidate] = struct{}{}
	if h.bridgeCancel == nil {
		bridgeCtx, cancel := context.WithCancel(context.Background())
		h.bridgeCancel = cancel
		go h.runBridge(bridgeCtx)
	}
	h.mu.Unlock()

	initial, err := h.capture(ctx)
	if err != nil {
		h.unregister(candidate)
		return err
	}
	h.publishTo(candidate, initial)
	return nil
}

func (h *Hub) unregister(candidate *client) {
	h.mu.Lock()
	if _, ok := h.clients[candidate]; !ok {
		h.mu.Unlock()
		return
	}
	delete(h.clients, candidate)
	candidate.cancel()
	var stop context.CancelFunc
	if len(h.clients) == 0 {
		stop = h.bridgeCancel
		h.bridgeCancel = nil
	}
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
}

func (h *Hub) runBridge(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-h.source.Updates():
			if !ok {
				return
			}
			snapshot, err := h.capture(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			h.publish(snapshot)
		}
	}
}

func (h *Hub) publish(snapshot queuedSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for candidate := range h.clients {
		h.offerLocked(candidate, snapshot)
	}
}

func (h *Hub) publishTo(candidate *client, snapshot queuedSnapshot) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[candidate]; !ok {
		return
	}
	h.offerLocked(candidate, snapshot)
}

func (h *Hub) offerLocked(candidate *client, snapshot queuedSnapshot) {
	if snapshot.sequence <= candidate.lastSequence {
		return
	}
	candidate.lastSequence = snapshot.sequence
	select {
	case candidate.queue <- snapshot:
		return
	default:
	}
	select {
	case <-candidate.queue:
	default:
	}
	select {
	case candidate.queue <- snapshot:
	default:
	}
}

func (h *Hub) clientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

// Close stops the bridge and disconnects every active client. It is safe to
// call more than once.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	stop := h.bridgeCancel
	h.bridgeCancel = nil
	clients := make([]*client, 0, len(h.clients))
	for candidate := range h.clients {
		clients = append(clients, candidate)
		delete(h.clients, candidate)
	}
	h.mu.Unlock()
	if stop != nil {
		stop()
	}
	for _, candidate := range clients {
		candidate.cancel()
	}
}
