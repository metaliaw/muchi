package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

type fakeStore struct {
	job  search.Job
	page search.ResultPage
}

func (s *fakeStore) CreateSearch(_ context.Context, _, _ string, input search.CreateInput) (search.Job, error) {
	s.job = search.Job{ID: "search_one", Status: search.JobQueued, Total: len(input.Cards), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return s.job, nil
}
func (s *fakeStore) GetSearch(context.Context, string) (search.Job, error) { return s.job, nil }
func (s *fakeStore) ListResults(_ context.Context, _ string, page search.ResultPage) (search.Result, error) {
	s.page = page
	return search.Result{SearchID: s.job.ID, Items: []search.Item{}, Cursor: page.After}, nil
}
func (s *fakeStore) CancelSearch(context.Context, string, string, string) (search.Job, error) {
	s.job.Status = search.JobCancelled
	return s.job, nil
}
func (*fakeStore) ClaimSearchItem(context.Context, string, time.Duration) (search.Item, error) {
	return search.Item{}, search.ErrNotFound
}
func (*fakeStore) RenewItemLease(context.Context, string, string, time.Duration) error  { return nil }
func (*fakeStore) CompleteSearchItem(context.Context, search.Item, []offer.Offer) error { return nil }
func (*fakeStore) CheckHealth(context.Context) error                                    { return nil }
func (*fakeStore) ListSourceHealth(context.Context) ([]search.SourceHealth, error)      { return nil, nil }

func TestCreateSearch(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Searches: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
	body := `{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Idempotency-Key", "request-one")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST /v1/searches status = %d body=%s", response.Code, response.Body.String())
	}
	var job search.Job
	if err := json.NewDecoder(response.Body).Decode(&job); err != nil || job.ID != "search_one" {
		t.Fatalf("POST /v1/searches job=%#v err=%v", job, err)
	}
}

func TestRequireHeaders(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Searches: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer secret")
	response = httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency status = %d", response.Code)
	}
}

func TestHideError(t *testing.T) {
	status, code, message := mapError(errors.New("database password leaked"))
	if status != http.StatusInternalServerError || code != "internal_error" || strings.Contains(message, "password") {
		t.Fatalf("mapError() status=%d code=%q message=%q", status, code, message)
	}
}

func TestListResultPage(t *testing.T) {
	store := &fakeStore{job: search.Job{ID: "search_one"}}
	api := API{Searches: search.Service{Searches: store}, Token: "secret"}
	request := httptest.NewRequest(http.MethodGet, "/v1/searches/search_one/results?after=12&limit=25", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.page != (search.ResultPage{After: 12, Limit: 25}) {
		t.Fatalf("result page=%#v status=%d body=%s", store.page, response.Code, response.Body.String())
	}
}

func TestRejectResultPage(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Searches: store}, Token: "secret"}
	request := httptest.NewRequest(http.MethodGet, "/v1/searches/search_one/results?after=-1&limit=101", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid result page status=%d body=%s", response.Code, response.Body.String())
	}
}
