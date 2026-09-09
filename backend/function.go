package muchiapi

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/httpapi"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

var (
	apiOnce    sync.Once
	apiHandler http.Handler
	apiError   error
	taskOnce   sync.Once
	taskWorker search.Worker
	taskError  error
	sweepOnce  sync.Once
	sweeper    search.Sweeper
	sweepError error
)

func init() {
	functions.HTTP("ServeAPI", ServeAPI)
	functions.HTTP("ProcessSearch", ProcessSearch)
	functions.HTTP("SweepQueue", SweepQueue)
}

// ServeAPI Serves the public Muchi HTTP contract.
func ServeAPI(w http.ResponseWriter, r *http.Request) {
	apiOnce.Do(func() {
		settings, err := config.LoadConfig()
		if err != nil {
			apiError = err
			return
		}
		runtime, err := application.BuildRuntime(r.Context(), settings, buildLogger(), true)
		if err != nil {
			apiError = err
			return
		}
		api := httpapi.API{
			Searches: runtime.Service, Health: runtime.Store, Token: settings.APIToken, Logger: buildLogger(),
			HealthCheckTimeout: settings.HealthCheckTimeout,
		}
		apiHandler = api.BuildHandler()
	})
	if apiError != nil {
		buildLogger().Error("ServeAPI Startup Failed", "error", apiError)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	apiHandler.ServeHTTP(w, r)
}

// ProcessSearch Handles one private Cloud Task.
func ProcessSearch(w http.ResponseWriter, r *http.Request) {
	taskOnce.Do(func() {
		settings, err := config.LoadConfig()
		if err != nil {
			taskError = err
			return
		}
		runtime, err := application.BuildRuntime(r.Context(), settings, buildLogger(), false)
		if err != nil {
			taskError = err
			return
		}
		taskWorker = search.Worker{
			Store: runtime.Store, Service: runtime.Service,
			LeaseDuration: settings.LeaseDuration, StockCheckLimit: settings.StockCheckLimit,
		}
	})
	if taskError != nil {
		buildLogger().Error("ProcessSearch Startup Failed", "error", taskError)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	worker := taskWorker
	worker.Owner = r.Header.Get("X-CloudTasks-TaskName")
	err := worker.ProcessNext(r.Context())
	if err != nil && !errors.Is(err, search.ErrNotFound) {
		http.Error(w, "Task failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// SweepQueue Puts back the Wake-ups that silent Turns spent.
//
// A Task carries no Item, so a Turn that ends without working spends a
// Wake-up that nothing replaces. Run on a Schedule, this notices the Drift
// and repairs it; without it, an Item whose Task was spent waits forever.
func SweepQueue(w http.ResponseWriter, r *http.Request) {
	sweepOnce.Do(func() {
		settings, err := config.LoadConfig()
		if err != nil {
			sweepError = err
			return
		}
		runtime, err := application.BuildRuntime(r.Context(), settings, buildLogger(), true)
		if err != nil {
			sweepError = err
			return
		}
		if runtime.Queue == nil {
			sweepError = errors.New("sweep needs a Task Queue: MUCHI_TASK_URL is empty")
			return
		}
		sweeper = search.Sweeper{
			Items: runtime.Store, Queue: runtime.Queue,
			Logger: buildLogger(), MaxWakes: settings.SweepMaxWakes,
		}
	})
	if sweepError != nil {
		buildLogger().Error("SweepQueue Startup Failed", "error", sweepError)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		return
	}
	woken, err := sweeper.SweepQueue(r.Context())
	if err != nil {
		http.Error(w, "Sweep failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"woken":%d}`, woken)
}

func buildLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, nil))
}
