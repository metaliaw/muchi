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
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

type API struct {
	Searches search.Service
	Health   search.HealthStore
	Token    string
	Logger   *slog.Logger
}

type errorReply struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	RequestID  string `json:"request_id"`
	IncidentID string `json:"incident_id,omitempty"`
}

type contextKey string

const requestIDKey contextKey = "request_id"

func (a API) Handler() http.Handler {
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
	var input search.CreateInput
	if err := decodeJSON(r, &input); err != nil {
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

func (a API) getSearch(w http.ResponseWriter, r *http.Request) {
	job, err := a.Searches.GetSearch(r.Context(), r.PathValue("search_id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (a API) listResults(w http.ResponseWriter, r *http.Request) {
	result, err := a.Searches.ListResults(r.Context(), r.PathValue("search_id"))
	if err != nil {
		a.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
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
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
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
		valid := len(value) == len(a.Token) && subtle.ConstantTimeCompare([]byte(value), []byte(a.Token)) == 1
		if !valid || a.Token == "" {
			writeJSON(w, http.StatusUnauthorized, errorReply{Code: "unauthorized", Message: "Bearer token required", RequestID: requestID(r)})
			return
		}
		next.ServeHTTP(w, r)
	})
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
				loggerOrDefault(a.Logger).ErrorContext(r.Context(), "Request Failed", "request_id", requestID(r), "incident_id", incident, "error", value)
				writeJSON(w, http.StatusInternalServerError, errorReply{Code: "internal_error", Message: "Internal service error", RequestID: requestID(r), IncidentID: incident})
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
		loggerOrDefault(a.Logger).ErrorContext(r.Context(), "Request Failed", "request_id", requestID(r), "incident_id", incident, "error", err)
	}
	writeJSON(w, status, errorReply{Code: code, Message: message, RequestID: requestID(r), IncidentID: incident})
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
		writeJSON(w, http.StatusBadRequest, errorReply{Code: "missing_idempotency_key", Message: "Valid Idempotency-Key header required", RequestID: requestID(r)})
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

func requestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey).(string)
	return id
}

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger != nil {
		return logger
	}
	return slog.Default()
}
