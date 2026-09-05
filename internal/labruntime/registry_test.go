package labruntime

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/session"
	"omenpath-lab/testutil"
)

func registryFixture(t *testing.T) (*Registry, *persistence.Store, *session.Service) {
	t.Helper()
	ctx := context.Background()
	cfg := config.Default()
	c := testutil.NewFakeClock(testutil.BaseTime)
	store, err := persistence.Open(ctx, t.TempDir()+"/labs.sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	service, err := session.New(store, cfg, c, nil)
	require.NoError(t, err)
	registry, err := New(store, cfg, c)
	require.NoError(t, err)
	t.Cleanup(registry.Close)
	return registry, store, service
}
func TestRegistry_ConcurrentAcquireSingleRuntimeAndDistinctResources(t *testing.T) {
	r, _, service := registryFixture(t)
	ctx := context.Background()
	a, err := service.Resolve(ctx, "")
	require.NoError(t, err)
	b, err := service.Resolve(ctx, "")
	require.NoError(t, err)
	leases := make(chan *Lease, 32)
	errs := make(chan error, 32)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() { l, e := r.Acquire(ctx, a.LabID); leases <- l; errs <- e })
	}
	wg.Wait()
	close(leases)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var first *Runtime
	for l := range leases {
		if first == nil {
			first = l.Runtime
		}
		require.Same(t, first, l.Runtime)
		l.Release()
		l.Release()
	}
	other, err := r.Acquire(ctx, b.LabID)
	require.NoError(t, err)
	defer other.Release()
	require.NotSame(t, first.Manager, other.Runtime.Manager)
	require.NotSame(t, first.Hub, other.Runtime.Hub)
	require.NotEqual(t, first.Manager.Updates(), other.Runtime.Manager.Updates())
	randomPtr := func(rt *Runtime) uintptr {
		return reflect.ValueOf(rt.Manager).Elem().FieldByName("random").Elem().Pointer()
	}
	require.NotEqual(t, randomPtr(first), randomPtr(other.Runtime))
	require.NoError(t, r.TickAll(ctx))
	r.Close()
	r.Close()
	_, err = r.Acquire(ctx, a.LabID)
	require.ErrorIs(t, err, ErrClosed)
	other.Release()
}
func TestRegistry_DeleteIdleSerializesAcquireAndHonorsLiveLease(t *testing.T) {
	r, store, service := registryFixture(t)
	ctx := context.Background()
	a, err := service.Resolve(ctx, "")
	require.NoError(t, err)
	l, err := r.Acquire(ctx, a.LabID)
	require.NoError(t, err)
	deleted, err := r.DeleteIfIdle(a.LabID, func() error { t.Error("active lease must block deletion"); return nil })
	require.NoError(t, err)
	require.False(t, deleted)
	l.Release()
	entered, finish, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		_, err := r.DeleteIfIdle(a.LabID, func() error {
			close(entered)
			<-finish
			_, err := store.DeleteExpiredLab(ctx, a.LabID, testutil.BaseTime.Add(31*24*time.Hour))
			return err
		})
		done <- err
	}()
	<-entered
	waitCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	_, err = r.Acquire(waitCtx, a.LabID)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	// Other laboratories can still resolve/load while this entry is deleting.
	b, err := service.Resolve(ctx, "")
	require.NoError(t, err)
	other, err := r.Acquire(ctx, b.LabID)
	require.NoError(t, err)
	other.Release()
	close(finish)
	require.NoError(t, <-done)
	_, err = r.Acquire(ctx, a.LabID)
	require.Error(t, err, "deleted lab must never return stale cached state")
}
func TestRegistry_DeleteFailurePreservesRuntimeAndRenewalProtectsRows(t *testing.T) {
	r, store, service := registryFixture(t)
	ctx := context.Background()
	a, err := service.Resolve(ctx, "")
	require.NoError(t, err)
	l, err := r.Acquire(ctx, a.LabID)
	require.NoError(t, err)
	original := l.Runtime
	l.Release()
	failure := errors.New("delete failed")
	ok, err := r.DeleteIfIdle(a.LabID, func() error { return failure })
	require.False(t, ok)
	require.ErrorIs(t, err, failure)
	l, err = r.Acquire(ctx, a.LabID)
	require.NoError(t, err)
	require.Same(t, original, l.Runtime)
	l.Release()
	ok, err = r.DeleteIfIdle(a.LabID, func() error {
		deleted, err := store.DeleteExpiredLab(ctx, a.LabID, testutil.BaseTime)
		require.False(t, deleted)
		return err
	})
	require.NoError(t, err)
	require.True(t, ok)
	l, err = r.Acquire(ctx, a.LabID)
	require.NoError(t, err)
	require.NotSame(t, original, l.Runtime)
	l.Release()
}
