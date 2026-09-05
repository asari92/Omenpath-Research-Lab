package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"omenpath-lab/internal/persistence"
)

type fakeWorkerRegistry struct {
	mu       sync.Mutex
	ticks    int
	active   bool
	deletes  int
	attempts int
}

func (r *fakeWorkerRegistry) TickAll(context.Context) error {
	r.mu.Lock()
	r.ticks++
	r.mu.Unlock()
	return nil
}
func (r *fakeWorkerRegistry) DeleteIfIdle(_ persistence.LabID, remove func() error) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts++
	if r.active {
		return false, nil
	}
	r.deletes++
	return true, remove()
}

type fakeCleanupStore struct {
	mu      sync.Mutex
	now     time.Time
	deleted []persistence.LabID
	failure error
}

func (s *fakeCleanupStore) ExpiredLabs(_ context.Context, now time.Time) ([]persistence.LabID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.now = now
	return []persistence.LabID{"00000000000000000000000000000001"}, s.failure
}
func (s *fakeCleanupStore) DeleteExpiredLab(_ context.Context, id persistence.LabID, now time.Time) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, id)
	return true, nil
}
func TestWorkers_TicksAndConditionalCleanupRespectRuntimeLease(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := &fakeWorkerRegistry{active: true}
	s := &fakeCleanupStore{}
	ticks, cleanup := make(chan time.Time), make(chan time.Time)
	done := make(chan error, 1)
	go func() { done <- runWorkers(ctx, r, s, ticks, cleanup) }()
	ticks <- time.Now()
	cleanup <- time.Now()
	require.Eventually(t, func() bool { r.mu.Lock(); defer r.mu.Unlock(); return r.attempts == 1 }, time.Second, time.Millisecond)
	r.mu.Lock()
	r.active = false
	r.mu.Unlock()
	cleanup <- time.Now()
	require.Eventually(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return len(s.deleted) > 0 }, time.Second, time.Millisecond)
	cancel()
	require.NoError(t, <-done)
	r.mu.Lock()
	require.Equal(t, 1, r.ticks)
	require.Equal(t, 1, r.deletes)
	r.mu.Unlock()
}
func TestWorkers_ReportsCleanupFailure(t *testing.T) {
	failure := errors.New("cleanup failed")
	cleanup := make(chan time.Time, 1)
	cleanup <- time.Now()
	require.ErrorIs(t, runWorkers(context.Background(), &fakeWorkerRegistry{}, &fakeCleanupStore{failure: failure}, make(chan time.Time), cleanup), failure)
}
func TestServerConfig_CookieSecureEnvironment(t *testing.T) {
	for _, tc := range []struct {
		env, secure string
		want, fail  bool
	}{{"", "", false, false}, {"", "true", true, false}, {"production", "", false, true}, {"production", "false", false, true}, {"production", "true", true, false}, {"", "garbage", false, true}} {
		cfg := serverConfigFromEnv(func(key string) string {
			if key == "OMENPATH_ENV" {
				return tc.env
			}
			if key == "OMENPATH_COOKIE_SECURE" {
				return tc.secure
			}
			return ""
		})
		require.Equal(t, tc.want, cfg.cookieSecure)
		require.Equal(t, tc.fail, cfg.configError != nil)
	}
}

func TestRunServices_ShutdownOrderHTTPWorkersHubsDatabase(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	var mu sync.Mutex
	order := []string{}
	appendOrder := func(s string) { mu.Lock(); order = append(order, s); mu.Unlock() }
	stopped := make(chan struct{})
	server := &fakeServingServer{listen: func() error { <-stopped; return nil }, shutdown: func(context.Context) error { appendOrder("http"); close(stopped); return nil }, close: func() error { return nil }}
	require.NoError(t, runServices(parent, server, func(ctx context.Context) error { <-ctx.Done(); appendOrder("workers"); return nil }, func() { appendOrder("hubs") }, func() error { appendOrder("database"); return nil }))
	require.Equal(t, []string{"http", "workers", "hubs", "database"}, order)
}
