package muchiapi

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cangrejometralleta/muchi-api/internal/gcp"
)

var (
	logger      = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	apiHost     = gcp.BuildLoader(logger, true)
	taskHost    = gcp.BuildLoader(logger, false)
	sweepHost   = gcp.BuildLoader(logger, true)
	releaseHost = gcp.BuildLoader(logger, false)
)

func init() {
	functions.HTTP("ServeAPI", ServeAPI)
	functions.HTTP("ProcessSearch", ProcessSearch)
	functions.HTTP("SweepQueue", SweepQueue)
	functions.HTTP("ReleaseOrders", ReleaseOrders)
}

// ServeAPI Serves the public Muchi HTTP contract.
func ServeAPI(w http.ResponseWriter, r *http.Request) {
	apiHost.ServeAPI(w, r)
}

// ProcessSearch Translates a private Cloud Task into one shared worker turn.
func ProcessSearch(w http.ResponseWriter, r *http.Request) {
	taskHost.ServeSearch(w, r, r.Header.Get("X-CloudTasks-TaskName"))
}

// SweepQueue Restores bounded execution opportunities for waiting work.
func SweepQueue(w http.ResponseWriter, r *http.Request) {
	sweepHost.ServeSweep(w, r)
}

// ReleaseOrders Moves expired pending orders through the shared operation.
func ReleaseOrders(w http.ResponseWriter, r *http.Request) {
	releaseHost.ServeRelease(w, r)
}
