package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"omenpath-lab/internal/clock"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/persistence"
	"omenpath-lab/internal/transport"
)

type Manager interface {
	State(context.Context) (persistence.Snapshot, error)
	Portal(context.Context, int64) (domain.Portal, []domain.Event, error)
	Events(context.Context) ([]domain.Event, error)
	Stabilize(context.Context, int64) error
	ClosePortal(context.Context, int64, bool) error
	SendObserver(context.Context, int64, bool) error
	RecallObserver(context.Context, int64, bool) error
	OpenExtraction(context.Context, int64) error
}

type API struct {
	manager Manager
	cfg     config.Config
	clock   clock.Clock
}

func NewRouter(manager Manager, cfg config.Config, clk clock.Clock) http.Handler {
	api := &API{manager: manager, cfg: cfg, clock: clk}
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
	return router
}

func (api *API) getState(w http.ResponseWriter, r *http.Request) {
	snapshot, err := api.manager.State(r.Context())
	if err != nil {
		writeInternal(w)
		return
	}
	view, err := transport.BuildStateSnapshot(snapshot, api.clock.Now(), api.cfg)
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
	_, history, err := api.manager.Portal(r.Context(), id)
	if err != nil {
		if errors.Is(err, engine.ErrPortalNotFound) {
			writeError(w, 404, "PORTAL_NOT_FOUND", "portal not found", false)
			return
		}
		writeInternal(w)
		return
	}
	snapshot, err := api.manager.State(r.Context())
	if err != nil {
		writeInternal(w)
		return
	}
	view, err := transport.BuildPortalDetails(snapshot, id, history, api.clock.Now(), api.cfg)
	if err != nil {
		writeInternal(w)
		return
	}
	writeJSON(w, 200, view)
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
