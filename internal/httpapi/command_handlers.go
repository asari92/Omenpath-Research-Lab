package httpapi

import (
	"bytes"
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
	body, err := decodePortalCommandBody(w, r)
	if err != nil {
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
	body, err := decodeExtractionBody(w, r)
	if err != nil || body.PlaneID <= 0 {
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

func decodeExactObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	opening, ok := token.(json.Delim)
	if !ok || opening != '{' {
		return nil, errors.New("request body must be a JSON object")
	}
	fields := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err = decoder.Token()
		if err != nil {
			return nil, err
		}
		name, ok := token.(string)
		if !ok {
			return nil, errors.New("request object field must be a string")
		}
		if _, duplicate := fields[name]; duplicate {
			return nil, errors.New("duplicate request field")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		fields[name] = value
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("request body must contain one JSON value")
	}
	return fields, nil
}

func decodePortalCommandBody(w http.ResponseWriter, r *http.Request) (portalCommandBody, error) {
	fields, err := decodeExactObject(w, r)
	if err != nil {
		return portalCommandBody{}, err
	}
	if len(fields) == 0 {
		return portalCommandBody{}, nil
	}
	raw, ok := fields["confirm"]
	if !ok || len(fields) != 1 {
		return portalCommandBody{}, errors.New("unknown request field")
	}
	var body portalCommandBody
	switch {
	case bytes.Equal(raw, []byte("true")):
		body.Confirm = true
	case bytes.Equal(raw, []byte("false")):
	default:
		return portalCommandBody{}, errors.New("confirm must be a boolean")
	}
	return body, nil
}

func decodeExtractionBody(w http.ResponseWriter, r *http.Request) (extractionBody, error) {
	fields, err := decodeExactObject(w, r)
	if err != nil {
		return extractionBody{}, err
	}
	raw, ok := fields["plane_id"]
	if !ok || len(fields) != 1 {
		return extractionBody{}, errors.New("plane_id is required")
	}
	var body extractionBody
	if err := json.Unmarshal(raw, &body.PlaneID); err != nil || bytes.Equal(raw, []byte("null")) {
		return extractionBody{}, errors.New("plane_id must be an integer")
	}
	return body, nil
}

func writeDomainError(w http.ResponseWriter, err error) {
	if hasCompositeCause(err) {
		writeInternal(w)
		return
	}
	switch {
	case errors.Is(err, engine.ErrPortalNotFound):
		writeError(w, http.StatusNotFound, "PORTAL_NOT_FOUND", "portal not found", false)
	case errors.Is(err, engine.ErrPlaneNotFound):
		writeError(w, http.StatusNotFound, "PLANE_NOT_FOUND", "plane not found", false)
	case isInvariantError(err):
		writeInternal(w)
	default:
		code, message, confirmable, ok := domainErrorDescriptor(err)
		if !ok {
			writeInternal(w)
			return
		}
		writeError(w, http.StatusConflict, code, message, confirmable)
	}
}

func hasCompositeCause(err error) bool {
	if err == nil {
		return false
	}
	if composite, ok := err.(interface{ Unwrap() []error }); ok {
		causes := composite.Unwrap()
		if len(causes) > 1 {
			return true
		}
		if len(causes) == 1 {
			return hasCompositeCause(causes[0])
		}
		return false
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return hasCompositeCause(wrapped.Unwrap())
	}
	return false
}

func isInvariantError(err error) bool {
	return errors.Is(err, domain.ErrObserverInvariant) ||
		errors.Is(err, domain.ErrLabEnergyInvariant) ||
		errors.Is(err, domain.ErrExtractionInvariant) ||
		errors.Is(err, domain.ErrSimulationInvariant) ||
		errors.Is(err, domain.ErrEventInvariant)
}

func domainErrorDescriptor(err error) (string, string, bool, bool) {
	known := []struct {
		err         error
		code        string
		confirmable bool
	}{
		{domain.ErrPortalNotOpen, "PORTAL_NOT_OPEN", false},
		{domain.ErrPortalAlreadyStable, "PORTAL_ALREADY_STABLE", false},
		{domain.ErrPortalOverchargeRisk, "PORTAL_OVERCHARGE_RISK", false},
		{domain.ErrConfirmationRequired, "CONFIRMATION_REQUIRED", true},
		{domain.ErrNoFreePortalSlot, "NO_FREE_PORTAL_SLOT", false},
		{domain.ErrObserverNotAvailable, "OBSERVER_NOT_AVAILABLE", false},
		{domain.ErrObserverNotWaitingReturn, "OBSERVER_NOT_WAITING_RETURN", false},
		{domain.ErrObserverLost, "OBSERVER_LOST", false},
		{domain.ErrPortalCriticalRisk, "PORTAL_CRITICAL_RISK", false},
		{domain.ErrPortalDirectionConflict, "PORTAL_DIRECTION_CONFLICT", false},
		{domain.ErrPortalBusy, "PORTAL_BUSY", false},
		{domain.ErrPortalCreaturesPresent, "PORTAL_CREATURES_PRESENT", false},
		{domain.ErrNoAvailableObserver, "NO_AVAILABLE_OBSERVER", false},
		{domain.ErrNoWaitingObserver, "NO_WAITING_OBSERVER", false},
		{domain.ErrInsufficientLabEnergy, "INSUFFICIENT_LAB_ENERGY", false},
		{domain.ErrExtractionSynchronizing, "EXTRACTION_SYNCHRONIZING", false},
	}
	for _, candidate := range known {
		if errors.Is(err, candidate.err) {
			return candidate.code, candidate.err.Error(), candidate.confirmable, true
		}
	}
	return "", "", false, false
}
