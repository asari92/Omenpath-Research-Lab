package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/testutil"
)

type blockingCommitRepository struct {
	*fakeRepository
	entered     chan struct{}
	release     chan struct{}
	enteredOnce sync.Once
	releaseOnce sync.Once
}

func newBlockingCommitRepository(snapshot persistence.Snapshot) *blockingCommitRepository {
	return &blockingCommitRepository{
		fakeRepository: newFakeRepository(snapshot),
		entered:        make(chan struct{}),
		release:        make(chan struct{}),
	}
}

func (r *blockingCommitRepository) Commit(ctx context.Context, snapshot persistence.Snapshot, drafts []domain.EventDraft) ([]domain.Event, error) {
	r.enteredOnce.Do(func() { close(r.entered) })
	select {
	case <-r.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return r.fakeRepository.Commit(ctx, snapshot, drafts)
}

func (r *blockingCommitRepository) unblock() {
	r.releaseOnce.Do(func() { close(r.release) })
}

func newBlockingTestManager(t *testing.T, snapshot persistence.Snapshot, now time.Time) (*LabManager, *blockingCommitRepository) {
	t.Helper()
	repo := newBlockingCommitRepository(snapshot)
	manager, err := NewLabManager(
		context.Background(), config.Default(), testutil.NewFakeClock(now), &lockedMinimumRandom{}, repo,
	)
	require.NoError(t, err)
	return manager, repo
}

func TestManagerRun_ConsumesInjectedTicksUntilContextCancel(t *testing.T) {
	base := testutil.BaseTime
	manager, _ := newTestManager(t, managerSnapshot(base), base)
	ticks := make(chan time.Time)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- manager.Run(ctx, ticks) }()

	ticks <- base.Add(time.Second)
	select {
	case <-manager.Updates():
	case <-time.After(time.Second):
		t.Fatal("manager did not consume injected tick")
	}
	cancel()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("manager did not stop promptly after context cancellation")
	}
	require.Equal(t, base.Add(time.Second), *manager.snapshot.Simulation.LastTickAt)
}

func TestManagerOwnership_ContextCancellationInterruptsWaitingCalls(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(context.Context, *LabManager) error
	}{
		{
			name: "state",
			call: func(ctx context.Context, manager *LabManager) error {
				_, err := manager.State(ctx)
				return err
			},
		},
		{
			name: "command",
			call: func(ctx context.Context, manager *LabManager) error {
				return manager.ClosePortal(ctx, 2, true)
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := testutil.BaseTime
			snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
			snapshot = withManagerPortal(snapshot, managerPortal(2, 2, base))
			manager, repo := newBlockingTestManager(t, snapshot, base)
			firstDone := make(chan error, 1)
			go func() { firstDone <- manager.ClosePortal(context.Background(), 1, true) }()
			<-repo.entered

			ctx, cancel := context.WithCancel(context.Background())
			waitingDone := make(chan error, 1)
			go func() { waitingDone <- test.call(ctx, manager) }()
			cancel()

			select {
			case err := <-waitingDone:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(250 * time.Millisecond):
				repo.unblock()
				require.NoError(t, <-firstDone)
				<-waitingDone
				t.Fatal("canceled call remained blocked waiting for manager ownership")
			}
			repo.unblock()
			require.NoError(t, <-firstDone)
		})
	}
}

func TestManagerRun_CancelAfterTickSelectionInterruptsOwnershipWait(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	manager, repo := newBlockingTestManager(t, snapshot, base)
	firstDone := make(chan error, 1)
	go func() { firstDone <- manager.ClosePortal(context.Background(), 1, true) }()
	<-repo.entered

	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(ctx, ticks) }()
	tickSelected := make(chan struct{})
	go func() {
		ticks <- base.Add(time.Second)
		close(tickSelected)
	}()
	<-tickSelected
	cancel()

	select {
	case err := <-runDone:
		require.NoError(t, err, "Run keeps its nil-on-cancel contract")
	case <-time.After(250 * time.Millisecond):
		repo.unblock()
		require.NoError(t, <-firstDone)
		<-runDone
		t.Fatal("Run remained blocked after cancellation of an already selected tick")
	}
	repo.unblock()
	require.NoError(t, <-firstDone)
}

func TestManagerRun_CancellationDoesNotHideRandomRestoreFailure(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	dueAt := base.Add(time.Second)
	snapshot.Simulation.NaturalSpawn.DueAt = &dueAt
	repo := newBlockingCommitRepository(snapshot)
	restoreFailure := errors.New("checkpoint restore unavailable")
	rnd := &checkpointSequenceRandom{
		ints:       []int{0, 20, 0, 10},
		floats:     []float64{50, 0.5, 0.9},
		restoreErr: restoreFailure,
	}
	manager, err := NewLabManager(
		context.Background(), config.Default(), testutil.NewFakeClock(base), rnd, repo,
	)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	runDone := make(chan error, 1)
	go func() { runDone <- manager.Run(ctx, ticks) }()
	ticks <- base.Add(2 * time.Second)
	<-repo.entered
	cancel()

	err = <-runDone
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, err, restoreFailure, "Run must surface failure of the random rollback")
}

func TestManagerRun_StaleTickIsNoOpAndLaterTickRuns(t *testing.T) {
	base := testutil.BaseTime
	latest := base.Add(2 * time.Second)
	snapshot := managerSnapshot(base)
	snapshot.Simulation.LastTickAt = &latest
	manager, repo := newTestManager(t, snapshot, latest)
	ticks := make(chan time.Time, 2)
	ticks <- base.Add(time.Second)
	ticks <- base.Add(3 * time.Second)
	close(ticks)

	err := manager.Run(context.Background(), ticks)

	require.NoError(t, err)
	require.Equal(t, base.Add(3*time.Second), *manager.snapshot.Simulation.LastTickAt)
	require.Empty(t, repo.commits)
	require.Len(t, manager.updates, 1, "only the later valid tick may signal")
}

func TestManagerTick_StaleClockIsNoOpWithoutSignal(t *testing.T) {
	base := testutil.BaseTime
	latest := base.Add(2 * time.Second)
	snapshot := managerSnapshot(base)
	snapshot.Simulation.LastTickAt = &latest
	manager, repo := newTestManager(t, snapshot, base.Add(time.Second))
	want := cloneTestSnapshot(manager.snapshot)

	err := manager.Tick(context.Background())

	require.NoError(t, err)
	require.Equal(t, want, manager.snapshot)
	require.Empty(t, repo.commits)
	select {
	case <-manager.Updates():
		t.Fatal("stale tick must not signal")
	default:
	}
}

func TestManagerTick_StaleClockDoesNotHideOtherSimulationInvariant(t *testing.T) {
	base := testutil.BaseTime
	latest := base.Add(2 * time.Second)
	snapshot := managerSnapshot(base)
	snapshot.Simulation.LastTickAt = &latest
	snapshot.Simulation.Planes = snapshot.Simulation.Planes[:84]
	manager, _ := newTestManager(t, snapshot, base.Add(time.Second))

	err := manager.Tick(context.Background())

	require.ErrorIs(t, err, domain.ErrSimulationInvariant)
}

func TestManagerTick_SignalsEvenWithoutMeaningfulDatabaseWrite(t *testing.T) {
	base := testutil.BaseTime
	clk := testutil.NewFakeClock(base)
	repo := newFakeRepository(managerSnapshot(base))
	manager, err := NewLabManager(context.Background(), config.Default(), clk, &lockedMinimumRandom{}, repo)
	require.NoError(t, err)
	clk.Advance(time.Second)

	require.NoError(t, manager.Tick(context.Background()))

	require.Empty(t, repo.commits)
	select {
	case <-manager.Updates():
	default:
		t.Fatal("expected one update signal")
	}
}

func TestManagerAction_SignalsAfterSuccessAndRejection(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	manager, _ := newTestManager(t, snapshot, base)

	require.NoError(t, manager.ClosePortal(context.Background(), 1, true))
	requireUpdate(t, manager.Updates())
	require.ErrorIs(t, manager.ClosePortal(context.Background(), 1, true), domain.ErrPortalNotOpen)
	requireUpdate(t, manager.Updates())
}

func TestManagerUpdates_CoalesceWithoutBlocking(t *testing.T) {
	base := testutil.BaseTime
	clk := testutil.NewFakeClock(base)
	repo := newFakeRepository(managerSnapshot(base))
	manager, err := NewLabManager(context.Background(), config.Default(), clk, &lockedMinimumRandom{}, repo)
	require.NoError(t, err)

	for range 100 {
		clk.Advance(time.Second)
		require.NoError(t, manager.Tick(context.Background()))
	}

	require.Len(t, manager.updates, 1)
}

func TestManagerConcurrentCommands_OnlyOneTransitionWins(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	manager, repo := newTestManager(t, snapshot, base)
	errorsOut := runConcurrently(2, func() error {
		return manager.ClosePortal(context.Background(), 1, true)
	})

	require.Len(t, errorsOut, 2)
	successes := 0
	rejections := 0
	for _, err := range errorsOut {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrPortalNotOpen):
			rejections++
		default:
			t.Fatalf("unexpected command error: %v", err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, rejections)
	require.Equal(t, 1, countDraftType(repo.commits, domain.EventPortalClosed))
	require.Equal(t, 1, countDraftType(repo.commits, domain.EventActionRejected))
}

type observingClock struct {
	now    time.Time
	called chan struct{}
	once   sync.Once
}

func (c *observingClock) Now() time.Time {
	c.once.Do(func() { close(c.called) })
	return c.now
}

func TestManagerConcurrentTicks_DoNotDuplicateEvents(t *testing.T) {
	base := testutil.BaseTime
	portal := managerPortal(1, 1, base)
	portal.ScheduledCloseAt = base.Add(time.Second)
	snapshot := withManagerPortal(managerSnapshot(base), portal)
	repo := newFakeRepository(snapshot)
	clk := &observingClock{now: base.Add(2 * time.Second), called: make(chan struct{})}
	manager, err := NewLabManager(context.Background(), config.Default(), clk, &lockedMinimumRandom{}, repo)
	require.NoError(t, err)

	manager.mu.Lock()
	done := make(chan error, 1)
	go func() { done <- manager.Tick(context.Background()) }()
	clockReadOutsideLock := false
	select {
	case <-clk.called:
		clockReadOutsideLock = true
	case <-time.After(50 * time.Millisecond):
	}
	manager.mu.Unlock()
	require.NoError(t, <-done)
	require.False(t, clockReadOutsideLock, "tick time must be captured under the manager lock")

	errorsOut := runConcurrently(16, func() error { return manager.Tick(context.Background()) })
	for _, err := range errorsOut {
		require.NoError(t, err)
	}
	require.Equal(t, 1, countDraftType(repo.commits, domain.EventPortalClosed))
}

func TestManagerConcurrentOpenings_KeepUniqueIDsAndSlots(t *testing.T) {
	base := testutil.BaseTime
	snapshot := managerSnapshot(base)
	makeWaitingObserver(&snapshot.Simulation.Observers[0], 1, base.Add(-10*time.Second), base)
	makeWaitingObserver(&snapshot.Simulation.Observers[1], 2, base.Add(-5*time.Second), base)
	manager, _ := newTestManager(t, snapshot, base)

	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i, planeID := range []int64{1, 2} {
		wg.Add(1)
		go func(index int, id int64) {
			defer wg.Done()
			errs[index] = manager.OpenExtraction(context.Background(), id)
		}(i, planeID)
	}
	wg.Wait()
	require.NoError(t, errs[0])
	require.NoError(t, errs[1])

	state := manager.snapshot.Simulation
	require.Len(t, state.Portals, 2)
	require.ElementsMatch(t, []int64{1, 2}, []int64{state.Portals[0].ID, state.Portals[1].ID})
	require.ElementsMatch(t, []int{1, 2}, []int{state.Portals[0].SlotIndex, state.Portals[1].SlotIndex})
	require.Equal(t, int64(3), state.NextPortalID)
}

func TestManagerConcurrentReadsAndWrites_ReturnConsistentSnapshots(t *testing.T) {
	base := testutil.BaseTime
	snapshot := withManagerPortal(managerSnapshot(base), managerPortal(1, 1, base))
	manager, _ := newTestManager(t, snapshot, base)

	start := make(chan struct{})
	results := make(chan error, 65)
	for range 32 {
		go func() {
			<-start
			state, err := manager.State(context.Background())
			if err == nil && (len(state.Simulation.Planes) != 85 || len(state.Simulation.Observers) != 10 || len(state.Simulation.Portals) != 1) {
				err = errors.New("torn snapshot")
			}
			results <- err
		}()
		go func() {
			<-start
			portal, _, err := manager.Portal(context.Background(), 1)
			if err == nil && portal.Status != domain.PortalStatusOpen && portal.Status != domain.PortalStatusClosed {
				err = errors.New("invalid portal status")
			}
			results <- err
		}()
	}
	go func() {
		<-start
		results <- manager.ClosePortal(context.Background(), 1, true)
	}()
	close(start)
	for range 65 {
		require.NoError(t, <-results)
	}
}

func requireUpdate(t *testing.T, updates <-chan struct{}) {
	t.Helper()
	select {
	case <-updates:
	default:
		t.Fatal("expected update signal")
	}
}

func runConcurrently(count int, operation func() error) []error {
	start := make(chan struct{})
	results := make(chan error, count)
	for range count {
		go func() {
			<-start
			results <- operation()
		}()
	}
	close(start)
	errorsOut := make([]error, count)
	for i := range errorsOut {
		errorsOut[i] = <-results
	}
	return errorsOut
}

func countDraftType(commits []repositoryCommit, eventType domain.EventType) int {
	count := 0
	for _, commit := range commits {
		for _, draft := range commit.drafts {
			if draft.EventType == eventType {
				count++
			}
		}
	}
	return count
}

func makeWaitingObserver(observer *domain.Observer, planeID int64, waitingSince, now time.Time) {
	observer.Status = domain.ObserverWaitingReturn
	observer.CurrentPlaneID = &planeID
	observer.ActivePortalID = nil
	observer.PhaseStartedAt = &waitingSince
	observer.PhaseEndsAt = nil
	observer.UpdatedAt = now
}
