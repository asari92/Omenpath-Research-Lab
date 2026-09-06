package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/httpapi"
	"omenpath-lab/internal/labruntime"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/session"
)

type serverConfig struct {
	databasePath string
	address      string
	webRoot      string
	cookieSecure bool
	configError  error
}

func serverConfigFromEnv(getenv func(string) string) serverConfig {
	result := serverConfig{databasePath: "./omenpath.db", address: ":8080"}
	if value := getenv("OMENPATH_DB_PATH"); value != "" {
		result.databasePath = value
	}
	if value := getenv("OMENPATH_ADDR"); value != "" {
		result.address = value
	}
	result.webRoot = getenv("OMENPATH_WEB_ROOT")
	result.cookieSecure, result.configError = config.CookieSecure(getenv("OMENPATH_COOKIE_SECURE"), getenv("OMENPATH_ENV") == "production")
	return result
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, serverConfigFromEnv(os.Getenv)); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, serverCfg serverConfig) error {
	if serverCfg.configError != nil {
		return serverCfg.configError
	}
	cfg := config.Default()
	store, err := persistence.Open(ctx, serverCfg.databasePath)
	if err != nil {
		return err
	}
	storeOwnedByServices := false
	defer func() {
		if !storeOwnedByServices {
			_ = store.Close()
		}
	}()
	if err := store.Migrate(ctx); err != nil {
		return err
	}
	registry, err := labruntime.New(store, cfg, clock.RealClock{})
	if err != nil {
		return err
	}
	defer registry.Close()
	resolver, err := session.New(store, cfg, clock.RealClock{}, nil)
	if err != nil {
		return err
	}
	router, err := httpapi.NewRouter(resolver, registry, cfg, serverCfg.cookieSecure)
	if err != nil {
		return err
	}
	handler := http.Handler(router)
	if serverCfg.webRoot != "" {
		handler, err = httpapi.NewProductionHandler(router, serverCfg.webRoot)
		if err != nil {
			return err
		}
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(cfg.SessionCleanupInterval)
	defer cleanupTicker.Stop()
	server := &http.Server{Addr: serverCfg.address, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	storeOwnedByServices = true
	return runServices(
		ctx,
		server,
		func(runCtx context.Context) error {
			return runWorkers(runCtx, registry, store, ticker.C, cleanupTicker.C)
		},
		registry.Close,
		store.Close,
	)
}

type runtimeWorkers interface {
	TickAll(context.Context) error
	DeleteIfIdle(persistence.LabID, func() error) (bool, error)
}
type cleanupStore interface {
	ExpiredLabs(context.Context, time.Time) ([]persistence.LabID, error)
	DeleteExpiredLab(context.Context, persistence.LabID, time.Time) (bool, error)
}

func runWorkers(ctx context.Context, registry runtimeWorkers, store cleanupStore, ticks, cleanup <-chan time.Time) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ticks:
			if !ok {
				return nil
			}
			if err := registry.TickAll(ctx); err != nil {
				return err
			}
		case now, ok := <-cleanup:
			if !ok {
				return nil
			}
			ids, err := store.ExpiredLabs(ctx, now)
			if err != nil {
				return err
			}
			for _, id := range ids {
				if _, err := registry.DeleteIfIdle(id, func() error { _, err := store.DeleteExpiredLab(ctx, id, now); return err }); err != nil {
					return err
				}
			}
		}
	}
}

type servingServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

// runServices drains HTTP, stops and awaits workers, closes runtime hubs, then
// closes persistence. Hijacked WebSockets are drained by the hub closure.
func runServices(
	parent context.Context,
	server servingServer,
	runManager func(context.Context) error,
	closeRouter func(),
	closeStore func() error,
) error {
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	defer cancel()
	serverDone := make(chan error, 1)
	managerDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()
	go func() { managerDone <- runManager(ctx) }()

	var serverErr error
	var managerErr error
	serverFinished := false
	managerFinished := false
	select {
	case <-parent.Done():
	case err := <-serverDone:
		serverFinished = true
		serverErr = err
	case err := <-managerDone:
		managerFinished = true
		managerErr = err
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(shutdownCtx)
	shutdownCancel()
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	if !serverFinished {
		serverErr = <-serverDone
	}
	cancel()
	if !managerFinished {
		managerErr = <-managerDone
	}
	closeRouter()
	storeErr := closeStore()
	return errors.Join(
		unexpectedServiceError(serverErr),
		unexpectedServiceError(managerErr),
		shutdownErr,
		storeErr,
	)
}

func unexpectedServiceError(err error) error {
	if err == nil {
		return nil
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		unexpected := make([]error, 0, len(joined.Unwrap()))
		for _, cause := range joined.Unwrap() {
			if filtered := unexpectedServiceError(cause); filtered != nil {
				unexpected = append(unexpected, filtered)
			}
		}
		switch len(unexpected) {
		case 0:
			return nil
		case 1:
			return unexpected[0]
		default:
			return errors.Join(unexpected...)
		}
	}
	if errors.Is(err, http.ErrServerClosed) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if cause := errors.Unwrap(err); cause != nil {
			return unexpectedServiceError(cause)
		}
		return nil
	}
	return err
}
