package errorhandler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
)

// Description Names the Public Answer for one HTTP Failure.
type Description struct {
	Status  int
	Code    string
	Message string
}

var (
	unauthorized            = Description{http.StatusUnauthorized, "unauthorized", "Bearer token required"}
	missingIdempotencyKey   = Description{http.StatusBadRequest, "missing_idempotency_key", "Valid Idempotency-Key header required"}
	invalidRequest          = Description{http.StatusBadRequest, "invalid_request", "Request is invalid"}
	notFound                = Description{http.StatusNotFound, "not_found", "Search was not found"}
	missingStoreList        = Description{http.StatusNotFound, "not_found", "Store publishes no such list"}
	idempotencyConflict     = Description{http.StatusConflict, "idempotency_conflict", "Idempotency key was used with another request"}
	invalidState            = Description{http.StatusConflict, "invalid_state", "Search cannot be cancelled"}
	internal                = Description{http.StatusInternalServerError, "internal_error", "Internal service error"}
	unsupportedCardMetadata = Description{http.StatusNotFound, "not_found", "Card metadata is not supported for this game"}
	unsupportedAutocomplete = Description{http.StatusNotFound, "not_found", "Card autocomplete is not supported for this game"}
	missingCardName         = Description{http.StatusBadRequest, "invalid_request", "Card name is required"}
	orderNotFound           = Description{http.StatusNotFound, "not_found", "Order was not found"}
	orderNotSupported       = Description{http.StatusUnprocessableEntity, "order_not_supported", "This store cannot place an order yet"}
	orderConflict           = Description{http.StatusConflict, "order_conflict", "Order already moved past that status"}
)

func Unauthorized() Description            { return unauthorized }
func MissingIdempotencyKey() Description   { return missingIdempotencyKey }
func InvalidRequest() Description          { return invalidRequest }
func NotFound() Description                { return notFound }
func MissingStoreList() Description        { return missingStoreList }
func IdempotencyConflict() Description     { return idempotencyConflict }
func InvalidState() Description            { return invalidState }
func Internal() Description                { return internal }
func UnsupportedCardMetadata() Description { return unsupportedCardMetadata }
func UnsupportedAutocomplete() Description { return unsupportedAutocomplete }
func MissingCardName() Description         { return missingCardName }
func OrderNotFound() Description           { return orderNotFound }
func OrderNotSupported() Description       { return orderNotSupported }
func OrderConflict() Description           { return orderConflict }

type reply struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	IncidentID string `json:"incident_id,omitempty"`
}

type Handler struct {
	Logger *slog.Logger
}

// Write Maps a Private Failure to its Public HTTP Answer.
func (h Handler) Write(w http.ResponseWriter, r *http.Request, requestID string, failure error) {
	h.WriteDescription(w, r, requestID, Describe(failure), failure)
}

// WriteDescription Writes a Known Public Answer without exposing its cause.
func (h Handler) WriteDescription(w http.ResponseWriter, r *http.Request, requestID string, description Description, cause any) {
	incident := ""
	if description.Status >= http.StatusInternalServerError {
		incident = fmt.Sprintf("inc_%d", time.Now().UnixNano())
		logger := h.Logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.ErrorContext(r.Context(), "Request Failed", "request_id", requestID,
			"incident_id", incident, "error", cause)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(description.Status)
	_ = json.NewEncoder(w).Encode(reply{
		Code: description.Code, Message: description.Message,
		RequestID: requestID, IncidentID: incident,
	})
}

func Describe(failure error) Description {
	switch {
	case errors.Is(failure, search.ErrInvalid):
		return invalidRequest
	case errors.Is(failure, search.ErrNotFound):
		return notFound
	case errors.Is(failure, moxfield.ErrNoList):
		return missingStoreList
	case errors.Is(failure, search.ErrConflict):
		return idempotencyConflict
	case errors.Is(failure, search.ErrNotRunning):
		return invalidState
	case errors.Is(failure, model.ErrOrderNotFound):
		return orderNotFound
	case errors.Is(failure, model.ErrOrderNotSupported):
		return orderNotSupported
	case errors.Is(failure, model.ErrOrderConflict):
		return orderConflict
	default:
		return internal
	}
}
