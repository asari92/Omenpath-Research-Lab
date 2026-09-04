package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/realtime"
	"omenpath-lab/internal/transport"
)

type Manager interface {
	State(context.Context) (persistence.Snapshot, error)
	Updates() <-chan struct{}
	Portal(context.Context, int64) (domain.Portal, []domain.Event, error)
	PortalState(context.Context, int64) (persistence.Snapshot, []domain.Event, error)
	Events(context.Context) ([]domain.Event, error)
	Stabilize(context.Context, int64) error
	ClosePortal(context.Context, int64, bool) error
	SendObserver(context.Context, int64, bool) error
	RecallObserver(context.Context, int64, bool) error
	OpenExtraction(context.Context, int64) error
	StartTutorial(context.Context) error
	ResetTutorial(context.Context) error
	TutorialSignal(context.Context, domain.TutorialSignal, *int64) error
	StartLive(context.Context) error
}

type API struct {
	manager Manager
	cfg     config.Config
}

// Router owns both HTTP routing and the realtime resources created alongside
// it. Close must be called by the composition root during graceful shutdown.
type Router struct {
	handler http.Handler
	hub     *realtime.Hub
}

func (r *Router) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	r.handler.ServeHTTP(w, request)
}

// Close is idempotent and disconnects active WebSocket clients while stopping
// the manager update bridge.
func (r *Router) Close() {
	if r == nil || r.hub == nil {
		return
	}
	r.hub.Close()
}

func NewRouter(manager Manager, cfg config.Config) (*Router, error) {
	if managerIsNil(manager) {
		return nil, fmt.Errorf("new router: nil manager")
	}
	if err := validateRouterConfig(cfg); err != nil {
		return nil, err
	}
	api := &API{manager: manager, cfg: cfg}
	router := chi.NewRouter()
	router.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed", false)
	})
	router.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "not found", false)
	})
	router.Get("/api/state", api.getState)
	router.Get("/api/portals/{id}", api.getPortal)
	router.Get("/api/events", api.getEvents)
	router.Post("/api/portals/{id}/stabilize", api.stabilize)
	router.Post("/api/portals/{id}/close", api.closePortal)
	router.Post("/api/portals/{id}/send-observer", api.sendObserver)
	router.Post("/api/portals/{id}/recall-observer", api.recallObserver)
	router.Post("/api/extraction/open", api.openExtraction)
	router.Post("/api/tutorial/start", api.startTutorial)
	router.Post("/api/tutorial/reset", api.resetTutorial)
	router.Post("/api/tutorial/signal", api.tutorialSignal)
	router.Post("/api/live/start", api.startLive)
	hub, err := realtime.NewHub(manager, cfg)
	if err != nil {
		return nil, fmt.Errorf("new router: realtime: %w", err)
	}
	router.Handle("/ws/lab", hub)
	return &Router{handler: router, hub: hub}, nil
}

func managerIsNil(manager Manager) bool {
	if manager == nil {
		return true
	}
	value := reflect.ValueOf(manager)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func validateRouterConfig(cfg config.Config) error {
	if cfg.MaxActivePortals != 7 || cfg.CreatureTransit <= 0 || cfg.ObserverTransitMax <= 0 ||
		cfg.RiskSafeHorizon <= 0 || cfg.RiskInstabilityPenalty < 0 || cfg.LabEnergyMax <= 0 ||
		cfg.LabRegenPerSec < 0 || cfg.CloseCost < 0 || cfg.StabilizeCost < 0 ||
		cfg.StabilizeBoost < 0 || cfg.StabilizeMaxStartEnergy < 0 || cfg.EmergencyDuration <= 0 {
		return fmt.Errorf("new router: invalid config: %w", domain.ErrSimulationInvariant)
	}
	return nil
}

func (api *API) getState(w http.ResponseWriter, r *http.Request) {
	snapshot, err := api.manager.State(r.Context())
	if err != nil {
		writeInternal(w)
		return
	}
	resolvedAt, err := snapshotResolvedAt(snapshot)
	if err != nil {
		writeInternal(w)
		return
	}
	view, err := transport.BuildStateSnapshot(snapshot, resolvedAt, api.cfg)
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (api *API) getPortal(w http.ResponseWriter, r *http.Request) {
	id, ok := positiveID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, 400, "INVALID_PATH", "invalid portal id", false)
		return
	}
	snapshot, history, err := api.manager.PortalState(r.Context(), id)
	if err != nil {
		if errors.Is(err, engine.ErrPortalNotFound) {
			writeError(w, 404, "PORTAL_NOT_FOUND", "portal not found", false)
			return
		}
		writeInternal(w)
		return
	}
	resolvedAt, err := snapshotResolvedAt(snapshot)
	if err != nil {
		writeInternal(w)
		return
	}
	view, err := transport.BuildPortalDetails(snapshot, id, history, resolvedAt, api.cfg)
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, 200, view)
}

func snapshotResolvedAt(snapshot persistence.Snapshot) (time.Time, error) {
	if snapshot.Simulation.LastTickAt == nil || snapshot.Simulation.LastTickAt.IsZero() {
		return time.Time{}, domain.ErrSimulationInvariant
	}
	return snapshot.Simulation.LastTickAt.UTC(), nil
}

func (api *API) getEvents(w http.ResponseWriter, r *http.Request) {
	events, err := api.manager.Events(r.Context())
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, 200, transport.BuildEvents(events))
}

func positiveID(raw string) (int64, bool) {
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

type errorBody struct {
	Error apiError `json:"error"`
}
type apiError struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	Confirmable bool   `json:"confirmable"`
}

func writeInternal(w http.ResponseWriter) {
	writeError(w, 500, "INTERNAL_ERROR", "internal server error", false)
}
func writeError(w http.ResponseWriter, status int, code, message string, confirmable bool) {
	writeJSON(w, status, errorBody{Error: apiError{Code: code, Message: message, Confirmable: confirmable}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
