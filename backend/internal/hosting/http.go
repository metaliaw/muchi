package hosting

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

func (l *Loader) ServeAPI(w http.ResponseWriter, r *http.Request) {
	instance, err := l.LoadInstance()
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	instance.Operations.API.ServeHTTP(w, r)
}

// ServeSearch Translates one wake-up into one application turn.
func (l *Loader) ServeSearch(w http.ResponseWriter, r *http.Request, owner string) {
	instance, err := l.LoadInstance()
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	err = instance.Operations.ProcessSearch(r.Context(), owner)
	if err != nil && !errors.Is(err, search.ErrNotFound) {
		http.Error(w, "Task failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (l *Loader) ServeSweep(w http.ResponseWriter, r *http.Request) {
	instance, err := l.LoadInstance()
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	woken, err := instance.Operations.SweepQueue(r.Context())
	if errors.Is(err, application.ErrDispatcherMissing) {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	if err != nil {
		http.Error(w, "Sweep failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"woken":%d}`, woken)
}

func (l *Loader) ServeRelease(w http.ResponseWriter, r *http.Request) {
	instance, err := l.LoadInstance()
	if err != nil {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	released, err := instance.Operations.ReleaseOrders(r.Context())
	if err != nil {
		http.Error(w, "Release failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"released":%d}`, released)
}
