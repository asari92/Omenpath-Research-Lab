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
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/httpapi"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/random"
)

type serverConfig struct {
	databasePath string
	address      string
}

func serverConfigFromEnv(getenv func(string) string) serverConfig {
	result := serverConfig{databasePath: "./omenpath.db", address: ":8080"}
	if value := getenv("OMENPATH_DB_PATH"); value != "" {
		result.databasePath = value
	}
	if value := getenv("OMENPATH_ADDR"); value != "" {
		result.address = value
	}
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
	if err := store.Bootstrap(ctx, time.Now().UTC(), cfg); err != nil {
		return err
	}
	manager, err := engine.NewLabManager(ctx, cfg, clock.RealClock{}, random.NewRealRandom(), store)
	if err != nil {
		return err
	}
	router, err := httpapi.NewRouter(manager, cfg)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	server := &http.Server{Addr: serverCfg.address, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	storeOwnedByServices = true
	return runServices(
		ctx,
		server,
		func(runCtx context.Context) error { return manager.Run(runCtx, ticker.C) },
		router.Close,
		store.Close,
	)
}

type servingServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
	Close() error
}

// runServices gives every exit path the same ownership protocol: cancel the
// manager, stop and await HTTP, await the manager, close realtime resources,
// then close persistence.
func runServices(
	parent context.Context,
	server servingServer,
	runManager func(context.Context) error,
	closeRouter func(),
	closeStore func() error,
) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	serverDone := make(chan error, 1)
	managerDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()
	go func() { managerDone <- runManager(ctx) }()

	var primary error
	serverFinished := false
	managerFinished := false
	select {
	case <-parent.Done():
	case err := <-serverDone:
		serverFinished = true
		if !errors.Is(err, http.ErrServerClosed) {
			primary = err
		}
	case err := <-managerDone:
		managerFinished = true
		primary = err
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := server.Shutdown(shutdownCtx)
	shutdownCancel()
	if shutdownErr != nil {
		shutdownErr = errors.Join(shutdownErr, server.Close())
	}
	if !serverFinished {
		if err := <-serverDone; primary == nil && !errors.Is(err, http.ErrServerClosed) {
			primary = err
		}
	}
	if !managerFinished {
		if err := <-managerDone; primary == nil {
			primary = err
		}
	}
	closeRouter()
	storeErr := closeStore()
	return errors.Join(primary, shutdownErr, storeErr)
}
