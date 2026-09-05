package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/stretchr/testify/require"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/httpapi"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/random"
	"omenpath-lab/internal/transport"
)

type fakeServingServer struct {
	listen       func() error
	shutdown     func(context.Context) error
	close        func() error
	shutdownSeen chan struct{}
}

func (s *fakeServingServer) ListenAndServe() error { return s.listen() }
func (s *fakeServingServer) Shutdown(ctx context.Context) error {
	if s.shutdownSeen != nil {
		select {
		case <-s.shutdownSeen:
		default:
			close(s.shutdownSeen)
		}
	}
	return s.shutdown(ctx)
}
func (s *fakeServingServer) Close() error { return s.close() }

func TestServerConfigFromEnv_UsesSafeDefaultsAndOverrides(t *testing.T) {
	defaults := serverConfigFromEnv(func(string) string { return "" })
	require.Equal(t, "./omenpath.db", defaults.databasePath)
	require.Equal(t, ":8080", defaults.address)
	overrides := map[string]string{"OMENPATH_DB_PATH": "/tmp/lab.sqlite", "OMENPATH_ADDR": "127.0.0.1:9090"}
	got := serverConfigFromEnv(func(key string) string { return overrides[key] })
	require.Equal(t, "/tmp/lab.sqlite", got.databasePath)
	require.Equal(t, "127.0.0.1:9090", got.address)
}

func TestCompositionSmoke_SQLiteRESTAndWebSocketInitialSnapshot(t *testing.T) {
	ctx := context.Background()
	cfg := config.Default()
	store, err := persistence.Open(ctx, t.TempDir()+"/smoke.sqlite")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, store.Close()) })
	require.NoError(t, store.Migrate(ctx))
	repository, err := store.ForLab(transitionalLabID)
	require.NoError(t, err)
	require.NoError(t, repository.Bootstrap(ctx, time.Now().UTC(), cfg))
	manager, err := engine.NewLabManager(ctx, cfg, clock.RealClock{}, random.NewRealRandom(), repository)
	require.NoError(t, err)
	router, err := httpapi.NewRouter(manager, cfg)
	require.NoError(t, err)
	t.Cleanup(router.Close)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	response, err := server.Client().Get(server.URL + "/api/state")
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.NoError(t, response.Body.Close())

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/lab"
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	require.NoError(t, err)
	defer conn.CloseNow()
	readCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var snapshot transport.StateSnapshot
	require.NoError(t, wsjson.Read(readCtx, conn, &snapshot))
	require.Equal(t, domain.ModeTutorial, snapshot.App.Mode)
	require.Equal(t, domain.TutorialActionCompleteIntro, *snapshot.App.ExpectedAction)
	require.Len(t, snapshot.Slots, 7)
}

func TestRunServices_ServerExitCancelsAndWaitsManager(t *testing.T) {
	serverErr := errors.New("bind failed")
	managerDone := make(chan struct{})
	server := &fakeServingServer{
		listen:   func() error { return serverErr },
		shutdown: func(context.Context) error { return nil },
		close:    func() error { return nil },
	}
	err := runServices(context.Background(), server, func(ctx context.Context) error {
		<-ctx.Done()
		close(managerDone)
		return nil
	}, func() {}, func() error { return nil })
	require.ErrorIs(t, err, serverErr)
	select {
	case <-managerDone:
	default:
		t.Fatal("manager was not canceled and awaited")
	}
}

func TestRunServices_ManagerErrorShutsDownServer(t *testing.T) {
	managerErr := errors.New("tick failed")
	stopListen := make(chan struct{})
	server := &fakeServingServer{
		listen:   func() error { <-stopListen; return http.ErrServerClosed },
		shutdown: func(context.Context) error { close(stopListen); return nil },
		close:    func() error { return nil },
	}
	err := runServices(context.Background(), server, func(context.Context) error { return managerErr }, func() {}, func() error { return nil })
	require.ErrorIs(t, err, managerErr)
}

func TestRunServices_JoinsIndependentManagerAndServerFailures(t *testing.T) {
	serverErr := errors.New("independent server failure")
	managerErr := errors.New("independent manager failure")
	release := make(chan struct{})
	server := &fakeServingServer{
		listen:   func() error { <-release; return serverErr },
		shutdown: func(context.Context) error { return nil },
		close:    func() error { return nil },
	}
	close(release)

	err := runServices(context.Background(), server, func(context.Context) error {
		return managerErr
	}, func() {}, func() error { return nil })
	require.ErrorIs(t, err, serverErr)
	require.ErrorIs(t, err, managerErr)
}

func TestUnexpectedServiceError_FiltersPureExpectedTermination(t *testing.T) {
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, http.ErrServerClosed} {
		require.NoError(t, unexpectedServiceError(err))
	}
}

func TestUnexpectedServiceError_PreservesUnexpectedLeafBesideExpectedTermination(t *testing.T) {
	tests := []struct {
		name     string
		expected error
	}{
		{name: "context canceled", expected: context.Canceled},
		{name: "server closed", expected: http.ErrServerClosed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sentinel := errors.New("random checkpoint restore failed")
			restoreErr := fmt.Errorf("restore random state: %w", sentinel)
			got := unexpectedServiceError(errors.Join(tc.expected, restoreErr))
			require.ErrorIs(t, got, sentinel)
			require.NotErrorIs(t, got, tc.expected)
			require.Equal(t, 1, strings.Count(got.Error(), sentinel.Error()))
		})
	}
}

func TestUnexpectedServiceError_NestedJoinsKeepEveryUnexpectedLeafWithoutNoise(t *testing.T) {
	first := errors.New("first independent failure")
	second := errors.New("second independent failure")
	got := unexpectedServiceError(errors.Join(
		context.Canceled,
		errors.Join(http.ErrServerClosed, first),
		errors.Join(context.DeadlineExceeded, second),
	))
	require.ErrorIs(t, got, first)
	require.ErrorIs(t, got, second)
	require.NotErrorIs(t, got, context.Canceled)
	require.NotErrorIs(t, got, context.DeadlineExceeded)
	require.NotErrorIs(t, got, http.ErrServerClosed)
	require.Equal(t, 1, strings.Count(got.Error(), first.Error()))
	require.Equal(t, 1, strings.Count(got.Error(), second.Error()))
	require.NotContains(t, got.Error(), context.Canceled.Error())
	require.NotContains(t, got.Error(), context.DeadlineExceeded.Error())
	require.NotContains(t, got.Error(), http.ErrServerClosed.Error())
}

func TestRunServices_ContextCancellationStopsBoth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stopListen := make(chan struct{})
	managerDone := make(chan struct{})
	server := &fakeServingServer{
		listen:   func() error { <-stopListen; return http.ErrServerClosed },
		shutdown: func(context.Context) error { close(stopListen); return nil },
		close:    func() error { return nil },
	}
	cancel()
	require.NoError(t, runServices(ctx, server, func(ctx context.Context) error {
		<-ctx.Done()
		close(managerDone)
		return nil
	}, func() {}, func() error { return nil }))
	select {
	case <-managerDone:
	default:
		t.Fatal("manager was not stopped")
	}
}

func TestRunServices_ClosesRouterBeforeStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopListen := make(chan struct{})
	server := &fakeServingServer{
		listen:   func() error { <-stopListen; return http.ErrServerClosed },
		shutdown: func(context.Context) error { close(stopListen); return nil },
		close:    func() error { return nil },
	}
	order := make([]string, 0, 2)
	require.NoError(t, runServices(ctx, server, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	}, func() { order = append(order, "router") }, func() error {
		order = append(order, "store")
		return nil
	}))
	require.Equal(t, []string{"router", "store"}, order)
}
