package muchiapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/hosting"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

type entryVault struct {
	application.Vault
	owner        string
	failure      error
	releaseAge   time.Duration
	releaseLimit int
}

func (v *entryVault) ClaimSearchItem(_ context.Context, owner string, _ time.Duration) (model.Item, error) {
	v.owner = owner
	if v.failure != nil {
		return model.Item{}, v.failure
	}
	return model.Item{}, search.ErrNotFound
}
func (v *entryVault) CountWaitingItems(context.Context, int) (int, error) { return 2, v.failure }
func (v *entryVault) ReleaseExpiredOrders(_ context.Context, age time.Duration, limit int) (int, error) {
	v.releaseAge, v.releaseLimit = age, limit
	return 4, v.failure
}

type entryQueue struct {
	application.Dispatcher
	wakes int
}

func (q *entryQueue) WakeWorker(context.Context, string) error { q.wakes++; return nil }

func TestCloudTaskKeepsItsOwnerAndDeliveryStatuses(t *testing.T) {
	previous := taskHost
	t.Cleanup(func() { taskHost = previous })
	vault := &entryVault{}
	operations := application.BuildOperations(application.Runtime{Vault: vault}, config.Config{WorkerID: "fallback"}, nil)
	taskHost = &hosting.Loader{Build: func(context.Context) (hosting.Instance, error) {
		return hosting.Instance{Operations: operations}, nil
	}}
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("X-CloudTasks-TaskName", "projects/test/tasks/one")
	response := httptest.NewRecorder()
	ProcessSearch(response, request)
	if response.Code != http.StatusNoContent || vault.owner != "projects/test/tasks/one" {
		t.Fatalf("status=%d owner=%q", response.Code, vault.owner)
	}
	vault.failure = errors.New("database unavailable")
	response = httptest.NewRecorder()
	ProcessSearch(response, request)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("failure status=%d", response.Code)
	}
}

func TestSchedulerEntriesKeepResponseAndPolicy(t *testing.T) {
	previousSweep, previousRelease := sweepHost, releaseHost
	t.Cleanup(func() { sweepHost, releaseHost = previousSweep, previousRelease })
	vault, queue := &entryVault{}, &entryQueue{}
	operations := application.BuildOperations(application.Runtime{Vault: vault, Queue: queue}, config.Config{
		SweepMaxWakes: 3, OrderPendingTTL: 24 * time.Hour, OrderReleaseMaxWakes: 8,
	}, nil)
	loader := &hosting.Loader{Build: func(context.Context) (hosting.Instance, error) {
		return hosting.Instance{Operations: operations}, nil
	}}
	sweepHost, releaseHost = loader, loader
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	response := httptest.NewRecorder()
	SweepQueue(response, request)
	if response.Code != http.StatusOK || response.Body.String() != `{"woken":2}` || queue.wakes != 2 {
		t.Fatalf("sweep status=%d body=%s wakes=%d", response.Code, response.Body, queue.wakes)
	}
	response = httptest.NewRecorder()
	ReleaseOrders(response, request)
	if response.Code != http.StatusOK || response.Body.String() != `{"released":4}` ||
		vault.releaseAge != 24*time.Hour || vault.releaseLimit != 8 {
		t.Fatalf("release status=%d body=%s age=%s limit=%d", response.Code, response.Body, vault.releaseAge, vault.releaseLimit)
	}
	vault.failure = errors.New("database unavailable")
	for _, entry := range []func(http.ResponseWriter, *http.Request){SweepQueue, ReleaseOrders} {
		response = httptest.NewRecorder()
		entry(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("failure status=%d", response.Code)
		}
	}
}

func TestCloudEntryRetriesTransientStartupFailure(t *testing.T) {
	previous := apiHost
	t.Cleanup(func() { apiHost = previous })
	attempts := 0
	apiHost = &hosting.Loader{Build: func(context.Context) (hosting.Instance, error) {
		attempts++
		if attempts == 1 {
			return hosting.Instance{}, errors.New("startup unavailable")
		}
		return hosting.Instance{Operations: application.Operations{API: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})}}, nil
	}}
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusOK} {
		response := httptest.NewRecorder()
		ServeAPI(response, httptest.NewRequest(http.MethodGet, "/v1/health", nil))
		if response.Code != status {
			t.Fatalf("status=%d want=%d", response.Code, status)
		}
	}
}

func TestSweepWithoutDispatchKeepsUnavailableStatus(t *testing.T) {
	previous := sweepHost
	t.Cleanup(func() { sweepHost = previous })
	operations := application.BuildOperations(application.Runtime{}, config.Config{}, nil)
	sweepHost = &hosting.Loader{Build: func(context.Context) (hosting.Instance, error) {
		return hosting.Instance{Operations: operations}, nil
	}}
	response := httptest.NewRecorder()
	SweepQueue(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", response.Code)
	}
}
