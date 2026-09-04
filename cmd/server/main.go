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
	defer store.Close()
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
	defer router.Close()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	managerDone := make(chan error, 1)
	go func() { managerDone <- manager.Run(ctx, ticker.C) }()

	server := &http.Server{Addr: serverCfg.address, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	serverDone := make(chan error, 1)
	go func() { serverDone <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-managerDone; err != nil {
			return err
		}
		return nil
	case err := <-serverDone:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case err := <-managerDone:
		if err != nil {
			_ = server.Close()
			return err
		}
		return nil
	}
}
