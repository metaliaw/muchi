package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

type API struct {
	Searches           search.Service
	Health             search.HealthStore
	Token              string
	Logger             *slog.Logger
	HealthCheckTimeout time.Duration
}

type errorReply struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	IncidentID string `json:"incident_id,omitempty"`
}

type contextKey string

const requestIDKey contextKey = "request_id"

const (
	defaultResultLimit = 50
	maxResultLimit     = 100
)

// BuildHandler Connects each API Story to its public route.
func (a API) BuildHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/health", a.getHealth)
	mux.Handle("GET /v1/health/sources", a.authenticate(http.HandlerFunc(a.listSourceHealth)))
	mux.Handle("POST /v1/searches", a.authenticate(http.HandlerFunc(a.createSearch)))
	mux.Handle("GET /v1/searches/{search_id}", a.authenticate(http.HandlerFunc(a.getSearch)))
	mux.Handle("GET /v1/searches/{search_id}/results", a.authenticate(http.HandlerFunc(a.listResults)))
	mux.Handle("POST /v1/searches/{search_id}/cancel", a.authenticate(http.HandlerFunc(a.cancelSearch)))
	mux.Handle("GET /v1/cards/offers", a.authenticate(http.HandlerFunc(a.findCardOffers)))
	return a.identifyRequest(a.recoverPanic(mux))
}

func (a API) createSearch(w http.ResponseWriter, r *http.Request) {
	key, ok := requireIdempotency(w, r)
	if !ok {
		return
	}
	input, err := decodeSearch(r)
	if err != nil {
		a.writeError(w, r, search.ErrInvalid)
		return
	}
	job, err := a.Searches.CreateSearch(r.Context(), key, input)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func decodeSearch(r *http.Request) (search.CreateInput, error) {
	var input search.CreateInput
	err := decodeJSON(r, &input)

	return input, err
}

func (a API) getSearch(w http.ResponseWriter, r *http.Request) {
	job, err := a.Searches.GetSearch(r.Context(), r.PathValue("search_id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (a API) listResults(w http.ResponseWriter, r *http.Request) {
	page, err := decodeResultPage(r)
	if err != nil {
		a.writeError(w, r, search.ErrInvalid)
		return
	}
	result, err := a.Searches.ListResults(r.Context(), r.PathValue("search_id"), page)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func decodeResultPage(r *http.Request) (search.ResultPage, error) {
	after, err := readQueryNumber(r, "after", 0)
	if err != nil || after < 0 {
		return search.ResultPage{}, search.ErrInvalid
	}
	limit, err := readQueryNumber(r, "limit", defaultResultLimit)
	if err != nil || limit < 1 || limit > maxResultLimit {
		return search.ResultPage{}, search.ErrInvalid
	}
	return search.ResultPage{After: after, Limit: limit}, nil
}

func readQueryNumber(r *http.Request, name string, fallback int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func (a API) cancelSearch(w http.ResponseWriter, r *http.Request) {
	key, ok := requireIdempotency(w, r)
	if !ok {
		return
	}
	job, err := a.Searches.CancelSearch(r.Context(), r.PathValue("search_id"), key)
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (a API) findCardOffers(w http.ResponseWriter, r *http.Request) {
	items, err := a.Searches.FindCardOffers(r.Context(), r.URL.Query().Get("name"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": r.URL.Query().Get("name"), "offers": items})
}

func (a API) getHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), a.HealthCheckTimeout)
	defer cancel()
	if err := a.Health.CheckHealth(ctx); err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a API) listSourceHealth(w http.ResponseWriter, r *http.Request) {
	items, err := a.Health.ListSourceHealth(r.Context())
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": items})
}

func (a API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		sameLength := len(value) == len(a.Token)
		sameToken := subtle.ConstantTimeCompare([]byte(value), []byte(a.Token)) == 1
		valid := sameLength && sameToken
		if !valid || a.Token == "" {
			writeJSON(w, http.StatusUnauthorized, buildAuthError(r))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func buildAuthError(r *http.Request) errorReply {
	return errorReply{
		Code:      "unauthorized",
		Message:   "Bearer token required",
		RequestID: readRequestID(r),
	}
}

func (a API) identifyRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				incident := fmt.Sprintf("inc_%d", time.Now().UnixNano())
				logRequestFailure(a.Logger, r, incident, value)
				writeJSON(w, http.StatusInternalServerError, buildErrorReply(r, "internal_error", "Internal service error", incident))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a API) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := mapError(err)
	incident := ""
	if status >= 500 {
		incident = fmt.Sprintf("inc_%d", time.Now().UnixNano())
		logRequestFailure(a.Logger, r, incident, err)
	}
	writeJSON(w, status, buildErrorReply(r, code, message, incident))
}

func logRequestFailure(logger *slog.Logger, r *http.Request, incident string, failure any) {
	selectLogger(logger).ErrorContext(
		r.Context(), "Request Failed",
		"request_id", readRequestID(r),
		"incident_id", incident,
		"error", failure,
	)
}

func buildErrorReply(r *http.Request, code, message, incident string) errorReply {
	return errorReply{
		Code:       code,
		Message:    message,
		RequestID:  readRequestID(r),
		IncidentID: incident,
	}
}

func mapError(err error) (int, string, string) {
	switch {
	case errors.Is(err, search.ErrInvalid):
		return http.StatusBadRequest, "invalid_request", "Request is invalid"
	case errors.Is(err, search.ErrNotFound):
		return http.StatusNotFound, "not_found", "Search was not found"
	case errors.Is(err, search.ErrConflict):
		return http.StatusConflict, "idempotency_conflict", "Idempotency key was used with another request"
	case errors.Is(err, search.ErrNotRunning):
		return http.StatusConflict, "invalid_state", "Search cannot be cancelled"
	default:
		return http.StatusInternalServerError, "internal_error", "Internal service error"
	}
}

func requireIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		reply := buildErrorReply(r, "missing_idempotency_key", "Valid Idempotency-Key header required", "")
		writeJSON(w, http.StatusBadRequest, reply)
		return "", false
	}
	return key, true
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("request must contain one JSON value")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readRequestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey).(string)
	return id
}

func selectLogger(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}
