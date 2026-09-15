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

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

type fakeStore struct {
	job  search.Job
	page search.ResultPage
}

type fakeMetadata struct{}

func (fakeMetadata) CardMetadata(_ context.Context, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	return cardmetadata.Metadata{Name: request.Name, Image: "https://images.example/pikachu.jpg"}, nil
}

func (fakeMetadata) Autocomplete(_ context.Context, name, _ string) ([]string, error) {
	return []string{name, name + " VMAX"}, nil
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
	body := `{"game":"magic","cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true}}`
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

func TestDecodeSearchDefaultsToMagic(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"cards":[{"name":"Sol Ring","quantity":1}]}`))

	input, err := decodeSearch(request)

	if err != nil || input.Game != search.GameMagic {
		t.Fatalf("decodeSearch() game=%q err=%v", input.Game, err)
	}
}

func TestDecodeSearchKeepsExplicitGame(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"game":"pokemon","cards":[{"name":"Pikachu","quantity":1}]}`))

	input, err := decodeSearch(request)

	if err != nil || input.Game != search.GamePokemon {
		t.Fatalf("decodeSearch() game=%q err=%v", input.Game, err)
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

func TestListSupportedGames(t *testing.T) {
	api := API{
		Token: "secret",
		SupportedGames: map[search.Game]string{
			search.GamePokemon: "Pokémon",
			search.GameMagic:   "Magic: The Gathering",
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/supported-games", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	api.BuildHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"games":[{"name":"Magic: The Gathering","reference_key":"magic"},{"name":"Pokémon","reference_key":"pokemon"}]}`+"\n" {
		t.Fatalf("GET /v1/supported-games status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestGetCardMetadataUsesSelectedGame(t *testing.T) {
	api := API{
		CardMetadata: map[search.Game]cardmetadata.Provider{search.GamePokemon: fakeMetadata{}},
		Token:        "secret",
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/cards/metadata?game=pokemon&name=Pikachu", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	api.BuildHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"name":"Pikachu","image":"https://images.example/pikachu.jpg"}`+"\n" {
		t.Fatalf("GET /v1/cards/metadata status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAutocompleteCardsUsesSelectedGame(t *testing.T) {
	api := API{
		Autocomplete: map[search.Game]cardmetadata.AutocompleteProvider{search.GamePokemon: fakeMetadata{}},
		Token:        "secret",
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/cards/autocomplete?game=pokemon&name=Pikachu", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	api.BuildHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != `{"suggestions":["Pikachu","Pikachu VMAX"]}`+"\n" {
		t.Fatalf("GET /v1/cards/autocomplete status=%d body=%s", response.Code, response.Body.String())
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

// TestARetiredOptionIsRefused Covers the Contract, not just the Code: the
// Decoder Refuses Unknown Fields, so a Client still Sending `stores_only`
// Learns it in the Answer instead of Believing it was Honoured.
func TestARetiredOptionIsRefused(t *testing.T) {
	body := `{"game":"magic","cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true,"stores_only":true}}`
	store := &fakeStore{}
	api := API{Searches: search.Service{Searches: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Idempotency-Key", "retired-option")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("POST /v1/searches status = %d, want 400", response.Code)
	}
}
