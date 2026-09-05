package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/realtime"
)

// Existing transport/command tests inject their source only in test code.
// Production routes always acquire a session-bound runtime first.
type managerTestRouter struct {
	http.Handler
	hub *realtime.Hub
}

func (r *managerTestRouter) Close() { r.hub.Close() }
func newManagerRouter(manager Manager, cfg config.Config) (*managerTestRouter, error) {
	if dependencyIsNil(manager) {
		return nil, fmt.Errorf("nil manager")
	}
	if err := validateRouterConfig(cfg); err != nil {
		return nil, err
	}
	hub, err := realtime.NewHub(manager, cfg)
	if err != nil {
		return nil, err
	}
	routes := newRoutes(cfg)
	return &managerTestRouter{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		routes.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), scopeKey{}, &requestScope{manager: manager, hub: hub})))
	}), hub: hub}, nil
}
