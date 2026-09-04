// Package engine owns the concurrent application state and orchestrates the
// pure domain model with atomic persistence.
package engine

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/random"
)

// LabManager is the sole mutation boundary for the active laboratory state.
// The exclusive mutex deliberately covers domain resolution, random draws,
// repository commit and memory publication so exactly one transition wins.
type LabManager struct {
	mu       sync.Mutex
	cfg      config.Config
	clock    clock.Clock
	random   random.Random
	repo     Repository
	snapshot persistence.Snapshot
	updates  chan struct{}
}

// NewLabManager loads the already-bootstrapped snapshot. Database migration
// and bootstrap remain composition-root responsibilities.
func NewLabManager(
	ctx context.Context,
	cfg config.Config,
	clk clock.Clock,
	rnd random.Random,
	repo Repository,
) (*LabManager, error) {
	if clk == nil || rnd == nil || repo == nil {
		return nil, fmt.Errorf("new lab manager: nil dependency")
	}
	snapshot, err := repo.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("new lab manager: load snapshot: %w", err)
	}
	return &LabManager{
		cfg:      cfg,
		clock:    clk,
		random:   rnd,
		repo:     repo,
		snapshot: cloneSnapshot(snapshot),
		updates:  make(chan struct{}, 1),
	}, nil
}

// Tick resolves one full supplied-time simulation step.
func (m *LabManager) Tick(ctx context.Context) error {
	if m == nil {
		return fmt.Errorf("tick: nil manager")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resolveLocked(ctx, m.clock.Now().UTC(), true)
}

func (m *LabManager) tickAt(ctx context.Context, now time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.resolveLocked(ctx, now.UTC(), true)
}

// Run consumes externally supplied tick instants until cancellation or channel
// closure. This keeps tests deterministic and leaves ticker ownership at the
// composition root.
func (m *LabManager) Run(ctx context.Context, ticks <-chan time.Time) error {
	if m == nil {
		return fmt.Errorf("run: nil manager")
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case at, ok := <-ticks:
			if !ok {
				return nil
			}
			if err := m.tickAt(ctx, at.UTC()); err != nil {
				return err
			}
		}
	}
}

// State resolves lifecycle before returning a fully detached snapshot.
func (m *LabManager) State(ctx context.Context) (persistence.Snapshot, error) {
	if m == nil {
		return persistence.Snapshot{}, fmt.Errorf("state: nil manager")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.resolveLocked(ctx, m.clock.Now().UTC(), false); err != nil {
		return persistence.Snapshot{}, err
	}
	return cloneSnapshot(m.snapshot), nil
}

// Portal returns one detached portal and its history from the shared event
// source, after first resolving all due lifecycle transitions.
func (m *LabManager) Portal(ctx context.Context, id int64) (domain.Portal, []domain.Event, error) {
	if m == nil {
		return domain.Portal{}, nil, fmt.Errorf("portal: nil manager")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.resolveLocked(ctx, m.clock.Now().UTC(), false); err != nil {
		return domain.Portal{}, nil, err
	}
	index, ok := portalIndex(m.snapshot.Simulation.Portals, id)
	if !ok {
		return domain.Portal{}, nil, ErrPortalNotFound
	}
	portal := clonePortal(m.snapshot.Simulation.Portals[index])
	history, err := m.repo.ListEvents(ctx, &id)
	if err != nil {
		return domain.Portal{}, nil, fmt.Errorf("portal history: %w", err)
	}
	return portal, cloneEvents(history), nil
}

// Events reads the global chronological event stream. It has no derived state
// to catch up and therefore does not take the manager mutex.
func (m *LabManager) Events(ctx context.Context) ([]domain.Event, error) {
	if m == nil {
		return nil, fmt.Errorf("events: nil manager")
	}
	events, err := m.repo.ListEvents(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("events: %w", err)
	}
	return cloneEvents(events), nil
}

// Updates is a coalescing edge-trigger used by the realtime layer.
func (m *LabManager) Updates() <-chan struct{} {
	if m == nil {
		return nil
	}
	return m.updates
}

func (m *LabManager) resolveLocked(ctx context.Context, now time.Time, signal bool) error {
	before := cloneSnapshot(m.snapshot)
	working := cloneSnapshot(m.snapshot)
	result, err := working.Simulation.ResolveTick(now, m.random, m.cfg)
	if err != nil {
		return err
	}
	if !result.Changed {
		return nil
	}
	if meaningfulSnapshotChange(before, working) || len(result.Events) > 0 {
		if _, err := m.repo.Commit(ctx, cloneSnapshot(working), cloneDrafts(result.Events)); err != nil {
			return fmt.Errorf("commit resolved state: %w", err)
		}
	}
	m.snapshot = cloneSnapshot(working)
	if signal {
		m.signalLocked()
	}
	return nil
}

func meaningfulSnapshotChange(before, after persistence.Snapshot) bool {
	before = cloneSnapshot(before)
	after = cloneSnapshot(after)
	before.Simulation.LastTickAt = nil
	after.Simulation.LastTickAt = nil
	return !reflect.DeepEqual(before, after)
}

func (m *LabManager) signalLocked() {
	select {
	case m.updates <- struct{}{}:
	default:
	}
}

func cloneSnapshot(snapshot persistence.Snapshot) persistence.Snapshot {
	clone := snapshot
	if snapshot.Simulation.Portals != nil {
		clone.Simulation.Portals = make([]domain.Portal, len(snapshot.Simulation.Portals))
		copy(clone.Simulation.Portals, snapshot.Simulation.Portals)
	}
	if snapshot.Simulation.Planes != nil {
		clone.Simulation.Planes = make([]domain.Plane, len(snapshot.Simulation.Planes))
		copy(clone.Simulation.Planes, snapshot.Simulation.Planes)
	}
	if snapshot.Simulation.Observers != nil {
		clone.Simulation.Observers = make([]domain.Observer, len(snapshot.Simulation.Observers))
		copy(clone.Simulation.Observers, snapshot.Simulation.Observers)
	}
	clone.Simulation.Lab.LeylineOverrideUntil = cloneTime(snapshot.Simulation.Lab.LeylineOverrideUntil)
	clone.Simulation.NaturalSpawn.ScheduledAt = cloneTime(snapshot.Simulation.NaturalSpawn.ScheduledAt)
	clone.Simulation.NaturalSpawn.DueAt = cloneTime(snapshot.Simulation.NaturalSpawn.DueAt)
	clone.Simulation.LastTickAt = cloneTime(snapshot.Simulation.LastTickAt)
	for i := range clone.Simulation.Planes {
		if snapshot.Simulation.Planes[i].Aliases != nil {
			clone.Simulation.Planes[i].Aliases = make([]string, len(snapshot.Simulation.Planes[i].Aliases))
			copy(clone.Simulation.Planes[i].Aliases, snapshot.Simulation.Planes[i].Aliases)
		}
		clone.Simulation.Planes[i].ExploredAt = cloneTime(snapshot.Simulation.Planes[i].ExploredAt)
	}
	for i := range clone.Simulation.Portals {
		clone.Simulation.Portals[i] = clonePortal(snapshot.Simulation.Portals[i])
	}
	for i := range clone.Simulation.Observers {
		clone.Simulation.Observers[i] = cloneObserver(snapshot.Simulation.Observers[i])
	}
	return clone
}

func clonePortal(portal domain.Portal) domain.Portal {
	portal.InstabilityCollapseAt = cloneTime(portal.InstabilityCollapseAt)
	portal.ExtractionSynchronizedAt = cloneTime(portal.ExtractionSynchronizedAt)
	portal.ClosedAt = cloneTime(portal.ClosedAt)
	return portal
}

func cloneObserver(observer domain.Observer) domain.Observer {
	observer.CurrentPlaneID = cloneInt64(observer.CurrentPlaneID)
	observer.ActivePortalID = cloneInt64(observer.ActivePortalID)
	observer.PhaseStartedAt = cloneTime(observer.PhaseStartedAt)
	observer.PhaseEndsAt = cloneTime(observer.PhaseEndsAt)
	return observer
}

func cloneEvents(events []domain.Event) []domain.Event {
	if events == nil {
		return nil
	}
	clone := make([]domain.Event, len(events))
	copy(clone, events)
	for i := range clone {
		clone[i].PortalID = cloneInt64(events[i].PortalID)
		clone[i].ObserverID = cloneInt64(events[i].ObserverID)
		clone[i].PlaneID = cloneInt64(events[i].PlaneID)
	}
	return clone
}

func cloneDrafts(drafts []domain.EventDraft) []domain.EventDraft {
	if drafts == nil {
		return nil
	}
	clone := make([]domain.EventDraft, len(drafts))
	copy(clone, drafts)
	for i := range clone {
		clone[i].PortalID = cloneInt64(drafts[i].PortalID)
		clone[i].ObserverID = cloneInt64(drafts[i].ObserverID)
		clone[i].PlaneID = cloneInt64(drafts[i].PlaneID)
	}
	return clone
}

func cloneInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func portalIndex(portals []domain.Portal, id int64) (int, bool) {
	for i := range portals {
		if portals[i].ID == id {
			return i, true
		}
	}
	return -1, false
}

func planeIndex(planes []domain.Plane, id int64) (int, bool) {
	for i := range planes {
		if planes[i].ID == id {
			return i, true
		}
	}
	return -1, false
}
