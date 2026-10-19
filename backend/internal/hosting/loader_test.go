package hosting

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/application"
)

func TestStartupRetriesFailureAndReusesSuccess(t *testing.T) {
	attempts := 0
	failure := errors.New("temporarily unavailable")
	loader := Loader{Build: func(ctx context.Context) (Instance, error) {
		attempts++
		if ctx.Err() != nil {
			t.Fatal("startup context cancelled")
		}
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 30*time.Second {
			t.Fatal("startup has no bounded deadline")
		}
		if attempts == 1 {
			return Instance{}, failure
		}
		return Instance{Operations: application.Operations{API: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
		})}}, nil
	}}
	for _, status := range []int{http.StatusServiceUnavailable, http.StatusAccepted, http.StatusAccepted} {
		response := httptest.NewRecorder()
		loader.ServeAPI(response, httptest.NewRequest(http.MethodGet, "/", nil))
		if response.Code != status {
			t.Fatalf("status=%d want=%d", response.Code, status)
		}
	}
	if attempts != 2 {
		t.Fatalf("startup attempts=%d", attempts)
	}
}

func TestCancelledFirstRequestDoesNotPoisonStartup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	loader := Loader{Build: func(startup context.Context) (Instance, error) {
		attempts++
		if startup.Err() != nil {
			return Instance{}, startup.Err()
		}
		return Instance{Operations: application.Operations{API: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.Context().Err(); err != nil {
				w.WriteHeader(http.StatusRequestTimeout)
				return
			}
			w.WriteHeader(http.StatusOK)
		})}}, nil
	}}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	loader.ServeAPI(response, request)
	if response.Code != http.StatusRequestTimeout {
		t.Fatalf("cancelled status=%d", response.Code)
	}
	response = httptest.NewRecorder()
	loader.ServeAPI(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || attempts != 1 {
		t.Fatalf("next status=%d attempts=%d", response.Code, attempts)
	}
}

func TestConcurrentColdRequestsBuildOneInstance(t *testing.T) {
	var attempts atomic.Int32
	loader := Loader{Build: func(context.Context) (Instance, error) {
		attempts.Add(1)
		return Instance{}, nil
	}}
	var group sync.WaitGroup
	for range 20 {
		group.Go(func() {
			if _, err := loader.LoadInstance(); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if attempts.Load() != 1 {
		t.Fatalf("startup attempts=%d", attempts.Load())
	}
}

func TestClosingReleasesResourcesOnceAndAllowsRestart(t *testing.T) {
	opens, closes := 0, 0
	failure := errors.New("close failed")
	loader := Loader{Build: func(context.Context) (Instance, error) {
		opens++
		return Instance{Close: func() error { closes++; return failure }}, nil
	}}
	if _, err := loader.LoadInstance(); err != nil {
		t.Fatal(err)
	}
	if err := loader.CloseInstance(); !errors.Is(err, failure) {
		t.Fatalf("close error=%v", err)
	}
	if err := loader.CloseInstance(); err != nil {
		t.Fatal(err)
	}
	if _, err := loader.LoadInstance(); err != nil {
		t.Fatal(err)
	}
	if opens != 2 || closes != 1 {
		t.Fatalf("opens=%d closes=%d", opens, closes)
	}
}
