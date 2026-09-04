package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"omenpath-lab/internal/domain"
	"omenpath-lab/internal/engine"
	"omenpath-lab/internal/transport"
)

type portalCommandBody struct {
	Confirm bool `json:"confirm"`
}

type extractionBody struct {
	PlaneID int64 `json:"plane_id"`
}

func (api *API) stabilize(w http.ResponseWriter, r *http.Request) {
	api.portalCommand(w, r, func(id int64, _ bool) error { return api.manager.Stabilize(r.Context(), id) })
}

func (api *API) closePortal(w http.ResponseWriter, r *http.Request) {
	api.portalCommand(w, r, func(id int64, confirm bool) error { return api.manager.ClosePortal(r.Context(), id, confirm) })
}

func (api *API) sendObserver(w http.ResponseWriter, r *http.Request) {
	api.portalCommand(w, r, func(id int64, confirm bool) error { return api.manager.SendObserver(r.Context(), id, confirm) })
}

func (api *API) recallObserver(w http.ResponseWriter, r *http.Request) {
	api.portalCommand(w, r, func(id int64, confirm bool) error { return api.manager.RecallObserver(r.Context(), id, confirm) })
}

func (api *API) portalCommand(w http.ResponseWriter, r *http.Request, command func(int64, bool) error) {
	id, ok := positiveID(chi.URLParam(r, "id"))
	if !ok {
		writeError(w, http.StatusBadRequest, "INVALID_PATH", "invalid portal id", false)
		return
	}
	var body portalCommandBody
	if err := decodeStrictJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", false)
		return
	}
	if err := command(id, body.Confirm); err != nil {
		writeDomainError(w, err)
		return
	}
	api.writeFreshState(w, r)
}

func (api *API) openExtraction(w http.ResponseWriter, r *http.Request) {
	var body extractionBody
	if err := decodeStrictJSON(w, r, &body); err != nil || body.PlaneID <= 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body", false)
		return
	}
	if err := api.manager.OpenExtraction(r.Context(), body.PlaneID); err != nil {
		writeDomainError(w, err)
		return
	}
	api.writeFreshState(w, r)
}

func (api *API) writeFreshState(w http.ResponseWriter, r *http.Request) {
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

func decodeStrictJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, engine.ErrPortalNotFound):
		writeError(w, http.StatusNotFound, "PORTAL_NOT_FOUND", "portal not found", false)
	case errors.Is(err, engine.ErrPlaneNotFound):
		writeError(w, http.StatusNotFound, "PLANE_NOT_FOUND", "plane not found", false)
	case isInvariantError(err):
		writeInternal(w)
	default:
		code, ok := domainErrorCode(err)
		if !ok {
			writeInternal(w)
			return
		}
		writeError(w, http.StatusConflict, code, err.Error(), errors.Is(err, domain.ErrConfirmationRequired))
	}
}

func isInvariantError(err error) bool {
	return errors.Is(err, domain.ErrObserverInvariant) ||
		errors.Is(err, domain.ErrLabEnergyInvariant) ||
		errors.Is(err, domain.ErrExtractionInvariant) ||
		errors.Is(err, domain.ErrSimulationInvariant) ||
		errors.Is(err, domain.ErrEventInvariant)
}

func domainErrorCode(err error) (string, bool) {
	known := []struct {
		err  error
		code string
	}{
		{domain.ErrPortalNotOpen, "PORTAL_NOT_OPEN"},
		{domain.ErrPortalAlreadyStable, "PORTAL_ALREADY_STABLE"},
		{domain.ErrPortalOverchargeRisk, "PORTAL_OVERCHARGE_RISK"},
		{domain.ErrConfirmationRequired, "CONFIRMATION_REQUIRED"},
		{domain.ErrNoFreePortalSlot, "NO_FREE_PORTAL_SLOT"},
		{domain.ErrObserverNotAvailable, "OBSERVER_NOT_AVAILABLE"},
		{domain.ErrObserverNotWaitingReturn, "OBSERVER_NOT_WAITING_RETURN"},
		{domain.ErrObserverLost, "OBSERVER_LOST"},
		{domain.ErrPortalCriticalRisk, "PORTAL_CRITICAL_RISK"},
		{domain.ErrPortalDirectionConflict, "PORTAL_DIRECTION_CONFLICT"},
		{domain.ErrPortalBusy, "PORTAL_BUSY"},
		{domain.ErrPortalCreaturesPresent, "PORTAL_CREATURES_PRESENT"},
		{domain.ErrNoAvailableObserver, "NO_AVAILABLE_OBSERVER"},
		{domain.ErrNoWaitingObserver, "NO_WAITING_OBSERVER"},
		{domain.ErrInsufficientLabEnergy, "INSUFFICIENT_LAB_ENERGY"},
		{domain.ErrExtractionSynchronizing, "EXTRACTION_SYNCHRONIZING"},
	}
	for _, candidate := range known {
		if errors.Is(err, candidate.err) {
			return candidate.code, true
		}
	}
	return "", false
}
