package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

type repositoryCommit struct {
	snapshot persistence.Snapshot
	drafts   []domain.EventDraft
}

type fakeRepository struct {
	mu          sync.Mutex
	snapshot    persistence.Snapshot
	events      []domain.Event
	commits     []repositoryCommit
	commitErr   error
	loadErr     error
	listErr     error
	lastFilter  *int64
	nextEventID int64
}

func newFakeRepository(snapshot persistence.Snapshot) *fakeRepository {
	return &fakeRepository{snapshot: cloneTestSnapshot(snapshot), nextEventID: 1}
}

func (r *fakeRepository) Load(context.Context) (persistence.Snapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.loadErr != nil {
		return persistence.Snapshot{}, r.loadErr
	}
	return cloneTestSnapshot(r.snapshot), nil
}

func (r *fakeRepository) Commit(_ context.Context, snapshot persistence.Snapshot, drafts []domain.EventDraft) ([]domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.commitErr != nil {
		return nil, r.commitErr
	}
	storedDrafts := cloneTestDrafts(drafts)
	created := make([]domain.Event, len(storedDrafts))
	for i, draft := range storedDrafts {
		created[i] = domain.Event{
			ID:          r.nextEventID,
			EventType:   draft.EventType,
			PortalID:    cloneTestInt64(draft.PortalID),
			ObserverID:  cloneTestInt64(draft.ObserverID),
			PlaneID:     cloneTestInt64(draft.PlaneID),
			Message:     draft.Message,
			PayloadJSON: draft.PayloadJSON,
			CreatedAt:   draft.CreatedAt,
		}
		r.nextEventID++
	}
	r.snapshot = cloneTestSnapshot(snapshot)
	r.events = append(r.events, cloneTestEvents(created)...)
	r.commits = append(r.commits, repositoryCommit{
		snapshot: cloneTestSnapshot(snapshot),
		drafts:   storedDrafts,
	})
	return cloneTestEvents(created), nil
}

func (r *fakeRepository) ListEvents(_ context.Context, portalID *int64) ([]domain.Event, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listErr != nil {
		return nil, r.listErr
	}
	r.lastFilter = cloneTestInt64(portalID)
	events := r.events
	if portalID != nil {
		events = domain.PortalHistory(events, *portalID)
	}
	return cloneTestEvents(events), nil
}

type lockedMinimumRandom struct{ mu sync.Mutex }

func (r *lockedMinimumRandom) IntInclusive(min, _ int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return min
}

func (r *lockedMinimumRandom) FloatRange(min, _ float64) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return min
}

func managerSnapshot(now time.Time) persistence.Snapshot {
	planes := make([]domain.Plane, 85)
	for i := range planes {
		planes[i] = domain.Plane{
			ID:          int64(i + 1),
			Name:        fmt.Sprintf("Plane %02d", i+1),
			Aliases:     []string{},
			CatalogTier: "core",
		}
	}
	scheduledAt := now
	dueAt := now.Add(time.Hour)
	lastTickAt := now
	return persistence.Snapshot{
		Simulation: domain.SimulationState{
			Lab:          domain.LabState{EnergyBase: 100, EnergyBaseAt: now},
			Planes:       planes,
			Observers:    domain.NewObserverRoster(10, now),
			NextPortalID: 1,
			NaturalSpawn: domain.NaturalSpawnState{ScheduledAt: &scheduledAt, DueAt: &dueAt},
			LastTickAt:   &lastTickAt,
		},
		App: domain.AppState{Mode: domain.ModeLive, TutorialStep: 9},
	}
}

func managerPortal(id int64, slot int, now time.Time) domain.Portal {
	portal := testutil.NewPortalBuilder().Build()
	portal.ID = id
	portal.Name = fmt.Sprintf("Omenpath #%04d", id)
	portal.SlotIndex = slot
	portal.OpenedAt = now
	portal.CreatedAt = now
	portal.UpdatedAt = now
	portal.EnergyBaseAt = now
	portal.ScheduledCloseAt = now.Add(time.Minute)
	return portal
}

func withManagerPortal(snapshot persistence.Snapshot, portal domain.Portal) persistence.Snapshot {
	snapshot.Simulation.Portals = append(snapshot.Simulation.Portals, portal)
	if snapshot.Simulation.NextPortalID <= portal.ID {
		snapshot.Simulation.NextPortalID = portal.ID + 1
	}
	return snapshot
}

func newTestManager(t *testing.T, snapshot persistence.Snapshot, now time.Time) (*LabManager, *fakeRepository) {
	t.Helper()
	repo := newFakeRepository(snapshot)
	manager, err := NewLabManager(
		context.Background(),
		config.Default(),
		testutil.NewFakeClock(now),
		&lockedMinimumRandom{},
		repo,
	)
	require.NoError(t, err)
	return manager, repo
}

func TestNewLabManager_LoadsPersistedSnapshot(t *testing.T) {
	want := managerSnapshot(testutil.BaseTime)
	want.Simulation.Planes[0].Aliases = []string{"Alpha"}
	repo := newFakeRepository(want)

	manager, err := NewLabManager(
		context.Background(), config.Default(), testutil.NewFakeClock(testutil.BaseTime),
		&lockedMinimumRandom{}, repo,
	)

	require.NoError(t, err)
	require.Equal(t, want, manager.snapshot)
	repo.snapshot.Simulation.Planes[0].Aliases[0] = "mutated repository alias"
	require.Equal(t, "Alpha", manager.snapshot.Simulation.Planes[0].Aliases[0])
}

func TestManagerTick_CommitsStateAndEventsAtomically(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	portal := managerPortal(1, 1, base)
	portal.ScheduledCloseAt = base.Add(time.Second)
	snapshot = withManagerPortal(snapshot, portal)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))

	err := manager.Tick(context.Background())

	require.NoError(t, err)
	require.Len(t, repo.commits, 1)
	commit := repo.commits[0]
	require.Equal(t, domain.PortalStatusClosed, commit.snapshot.Simulation.Portals[0].Status)
	require.Equal(t, []domain.EventType{domain.EventPortalClosed}, draftTypes(commit.drafts))
	require.Equal(t, commit.snapshot, manager.snapshot)
}

func TestManagerTick_PersistenceFailureKeepsMemoryUnchanged(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	snapshot.Simulation.Portals[0].ScheduledCloseAt = base.Add(time.Second)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))
	want := cloneTestSnapshot(manager.snapshot)
	repo.commitErr = errors.New("database unavailable")

	err := manager.Tick(context.Background())

	require.ErrorContains(t, err, "database unavailable")
	require.Equal(t, want, manager.snapshot)
	require.Empty(t, repo.commits)
}

func TestManagerCommand_ResolvesWholeSimulationBeforeAction(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	target := managerPortal(1, 1, base)
	hidden := base.Add(time.Minute)
	target.Stability = domain.PortalUnstable
	target.InstabilityCollapseAt = &hidden
	target.EnergyBase = 50
	dueOther := managerPortal(2, 2, base)
	dueOther.ScheduledCloseAt = base.Add(time.Second)
	snapshot = withManagerPortal(withManagerPortal(snapshot, target), dueOther)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))

	err := manager.Stabilize(context.Background(), 1)

	require.NoError(t, err)
	require.Len(t, repo.commits, 1)
	commit := repo.commits[0]
	require.Equal(t, domain.PortalStable, commit.snapshot.Simulation.Portals[0].Stability)
	require.Equal(t, domain.PortalStatusClosed, commit.snapshot.Simulation.Portals[1].Status)
	require.ElementsMatch(t,
		[]domain.EventType{domain.EventPortalClosed, domain.EventPortalStabilized},
		draftTypes(commit.drafts),
	)
}

func TestManagerCommand_DueTerminalPortalWins(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	portal := managerPortal(1, 1, base)
	portal.ScheduledCloseAt = base.Add(time.Second)
	snapshot = withManagerPortal(snapshot, portal)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))

	err := manager.ClosePortal(context.Background(), 1, true)

	require.ErrorIs(t, err, domain.ErrPortalNotOpen)
	require.Len(t, repo.commits, 1)
	require.Equal(t, domain.PortalStatusClosed, repo.commits[0].snapshot.Simulation.Portals[0].Status)
	require.Equal(t,
		[]domain.EventType{domain.EventPortalClosed, domain.EventActionRejected},
		draftTypes(repo.commits[0].drafts),
	)
}

func TestManagerRejectedCommand_CommitsCatchupAndActionRejected(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	stableTarget := managerPortal(1, 1, base)
	dueOther := managerPortal(2, 2, base)
	dueOther.ScheduledCloseAt = base.Add(time.Second)
	snapshot = withManagerPortal(withManagerPortal(snapshot, stableTarget), dueOther)
	manager, repo := newTestManager(t, snapshot, base.Add(2*time.Second))

	err := manager.Stabilize(context.Background(), 1)

	require.ErrorIs(t, err, domain.ErrPortalAlreadyStable)
	require.Len(t, repo.commits, 1)
	require.Equal(t,
		[]domain.EventType{domain.EventPortalClosed, domain.EventActionRejected},
		draftTypes(repo.commits[0].drafts),
	)
	require.Equal(t, domain.PortalStatusClosed, manager.snapshot.Simulation.Portals[1].Status)
}

func TestManagerMissingEntity_CreatesActionRejectedWithRequestedID(t *testing.T) {
	manager, repo := newTestManager(t, managerSnapshot(testutil.BaseTime), testutil.BaseTime)

	err := manager.ClosePortal(context.Background(), 404, true)

	require.ErrorIs(t, err, ErrPortalNotFound)
	require.Len(t, repo.commits, 1)
	require.Len(t, repo.commits[0].drafts, 1)
	draft := repo.commits[0].drafts[0]
	require.Equal(t, domain.EventActionRejected, draft.EventType)
	require.Equal(t, int64(404), *draft.PortalID)
	var payload map[string]string
	require.NoError(t, json.Unmarshal([]byte(draft.PayloadJSON), &payload))
	require.Equal(t, "CLOSE", payload["action"])
}

func TestManagerMalformedTransportIsOutsideDomainBoundary(t *testing.T) {
	type malformedTransportRecorder interface {
		RejectMalformedTransport(context.Context, error) error
	}
	manager, _ := newTestManager(t, managerSnapshot(testutil.BaseTime), testutil.BaseTime)
	_, exposed := any(manager).(malformedTransportRecorder)
	require.False(t, exposed, "transport parsing failures must not enter LabManager")
}

func TestManagerState_ReturnsDeepCopy(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	snapshot.Simulation.Planes[0].Aliases = []string{"Original"}
	portal := managerPortal(1, 1, base)
	snapshot = withManagerPortal(snapshot, portal)
	manager, _ := newTestManager(t, snapshot, base)

	got, err := manager.State(context.Background())
	require.NoError(t, err)
	got.Simulation.Planes[0].Aliases[0] = "Changed"
	*got.Simulation.NaturalSpawn.DueAt = base
	got.Simulation.Portals[0].Name = "Changed"

	require.Equal(t, "Original", manager.snapshot.Simulation.Planes[0].Aliases[0])
	require.Equal(t, base.Add(time.Hour), *manager.snapshot.Simulation.NaturalSpawn.DueAt)
	require.Equal(t, "Omenpath #0001", manager.snapshot.Simulation.Portals[0].Name)
}

func TestManagerPortal_ReturnsFilteredSharedHistory(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	manager, repo := newTestManager(t, snapshot, base)
	portalID := int64(1)
	otherID := int64(2)
	repo.events = []domain.Event{
		{ID: 1, EventType: domain.EventPortalOpened, PortalID: &portalID, PlaneID: ptrInt64(1), Message: "one", PayloadJSON: `{}`, CreatedAt: base},
		{ID: 2, EventType: domain.EventPortalOpened, PortalID: &otherID, PlaneID: ptrInt64(2), Message: "two", PayloadJSON: `{}`, CreatedAt: base.Add(time.Second)},
	}
	repo.nextEventID = 3

	portal, history, err := manager.Portal(context.Background(), 1)

	require.NoError(t, err)
	require.Equal(t, int64(1), portal.ID)
	require.Equal(t, int64(1), *repo.lastFilter)
	require.Len(t, history, 1)
	require.Equal(t, "one", history[0].Message)
	*history[0].PortalID = 99
	require.Equal(t, int64(1), *repo.events[0].PortalID)
}

func draftTypes(drafts []domain.EventDraft) []domain.EventType {
	types := make([]domain.EventType, len(drafts))
	for i := range drafts {
		types[i] = drafts[i].EventType
	}
	return types
}

func ptrInt64(value int64) *int64 { return &value }

func cloneTestSnapshot(snapshot persistence.Snapshot) persistence.Snapshot {
	clone := snapshot
	clone.Simulation.Portals = append([]domain.Portal(nil), snapshot.Simulation.Portals...)
	clone.Simulation.Planes = append([]domain.Plane(nil), snapshot.Simulation.Planes...)
	clone.Simulation.Observers = append([]domain.Observer(nil), snapshot.Simulation.Observers...)
	clone.Simulation.Lab.LeylineOverrideUntil = cloneTestTime(snapshot.Simulation.Lab.LeylineOverrideUntil)
	clone.Simulation.NaturalSpawn.ScheduledAt = cloneTestTime(snapshot.Simulation.NaturalSpawn.ScheduledAt)
	clone.Simulation.NaturalSpawn.DueAt = cloneTestTime(snapshot.Simulation.NaturalSpawn.DueAt)
	clone.Simulation.LastTickAt = cloneTestTime(snapshot.Simulation.LastTickAt)
	for i := range clone.Simulation.Planes {
		clone.Simulation.Planes[i].Aliases = append([]string(nil), snapshot.Simulation.Planes[i].Aliases...)
		clone.Simulation.Planes[i].ExploredAt = cloneTestTime(snapshot.Simulation.Planes[i].ExploredAt)
	}
	for i := range clone.Simulation.Portals {
		clone.Simulation.Portals[i].InstabilityCollapseAt = cloneTestTime(snapshot.Simulation.Portals[i].InstabilityCollapseAt)
		clone.Simulation.Portals[i].ExtractionSynchronizedAt = cloneTestTime(snapshot.Simulation.Portals[i].ExtractionSynchronizedAt)
		clone.Simulation.Portals[i].ClosedAt = cloneTestTime(snapshot.Simulation.Portals[i].ClosedAt)
	}
	for i := range clone.Simulation.Observers {
		clone.Simulation.Observers[i].CurrentPlaneID = cloneTestInt64(snapshot.Simulation.Observers[i].CurrentPlaneID)
		clone.Simulation.Observers[i].ActivePortalID = cloneTestInt64(snapshot.Simulation.Observers[i].ActivePortalID)
		clone.Simulation.Observers[i].PhaseStartedAt = cloneTestTime(snapshot.Simulation.Observers[i].PhaseStartedAt)
		clone.Simulation.Observers[i].PhaseEndsAt = cloneTestTime(snapshot.Simulation.Observers[i].PhaseEndsAt)
	}
	return clone
}

func cloneTestDrafts(drafts []domain.EventDraft) []domain.EventDraft {
	clone := append([]domain.EventDraft(nil), drafts...)
	for i := range clone {
		clone[i].PortalID = cloneTestInt64(drafts[i].PortalID)
		clone[i].ObserverID = cloneTestInt64(drafts[i].ObserverID)
		clone[i].PlaneID = cloneTestInt64(drafts[i].PlaneID)
	}
	return clone
}

func cloneTestEvents(events []domain.Event) []domain.Event {
	clone := append([]domain.Event(nil), events...)
	for i := range clone {
		clone[i].PortalID = cloneTestInt64(events[i].PortalID)
		clone[i].ObserverID = cloneTestInt64(events[i].ObserverID)
		clone[i].PlaneID = cloneTestInt64(events[i].PlaneID)
	}
	return clone
}

func cloneTestInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneTestTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}
