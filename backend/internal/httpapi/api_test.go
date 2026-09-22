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
	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
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

func gamesAPI() API {
	return API{
		Token: "secret",
		SupportedGames: []stores.GameSupport{
			{Key: "magic", Name: "Magic: The Gathering", Singles: true, Sealed: true},
			{Key: "mitos-y-leyendas", Name: "Mitos y Leyendas", Singles: true},
			{Key: "sellado", Name: "Solo Cajas", Sealed: true},
		},
	}
}

func askGames(t *testing.T, query string) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/supported-games"+query, nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	gamesAPI().BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("GET /v1/supported-games%s status=%d", query, response.Code)
	}
	return response.Body.String()
}

func TestListSupportedGames(t *testing.T) {
	body := askGames(t, "")

	wanted := `{"games":[{"reference_key":"magic","name":"Magic: The Gathering","singles":true,"sealed":true},` +
		`{"reference_key":"mitos-y-leyendas","name":"Mitos y Leyendas","singles":true,"sealed":false},` +
		`{"reference_key":"sellado","name":"Solo Cajas","singles":false,"sealed":true}]}` + "\n"
	if body != wanted {
		t.Fatalf("body=%s", body)
	}
}

// Un Selector de Cajas Dibuja solo los Juegos que Tienen Cajas: Ofrecer el
// resto Prometería una Búsqueda que Contesta vacía se Pregunte como se Pregunte.
func TestListSupportedGamesByKind(t *testing.T) {
	if body := askGames(t, "?kind=sealed"); strings.Contains(body, "mitos-y-leyendas") {
		t.Fatalf("a singles-only game answered a sealed question: %s", body)
	}
	if body := askGames(t, "?kind=single"); strings.Contains(body, "Solo Cajas") {
		t.Fatalf("a sealed-only game answered a card question: %s", body)
	}
}

func TestListSupportedGamesRefusesAnUnknownKind(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/v1/supported-games?kind=barajas", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	gamesAPI().BuildHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", response.Code)
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

// stockStore Answers one Page with two Offers, one of each Price.
type stockStore struct{ fakeStore }

func (s *stockStore) ListResults(context.Context, string, search.ResultPage) (search.Result, error) {
	return search.Result{SearchID: "search_one", Items: []search.Item{{ID: "item-1", Offers: []offer.Offer{
		{ID: "cheap", URL: "https://store.test/cheap", PriceAmount: "1000"},
		{ID: "dear", URL: "https://store.test/dear", PriceAmount: "2000"},
	}}}}, nil
}

// countingStock Answers Sold Out for the cheap Offer and counts its Visits.
type countingStock struct{ visits []string }

func (c *countingStock) CheckStock(_ context.Context, item offer.Offer) (offer.StockReading, error) {
	c.visits = append(c.visits, item.ID)
	if item.ID == "cheap" {
		return offer.CountStock("unavailable", 0), nil
	}
	return offer.CountStock("available", 3), nil
}

func askStock(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	store := &stockStore{}
	api := API{Searches: search.Service{Searches: store, Stocks: &countingStock{}}, Token: "secret"}
	request := httptest.NewRequest(http.MethodPost, "/v1/searches/search_one/stock", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	recorder := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(recorder, request)
	return recorder
}

// TestCheckStockAnswersEachOfferAsked Spells the Contract the Caller Reads: one
// Reading per Offer, in the Order Asked, and a Count when the Store Kept one.
func TestCheckStockAnswersEachOfferAsked(t *testing.T) {
	recorder := askStock(t, `{"offers":["cheap","dear"]}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var reply struct {
		Offers []search.OfferStock `json:"offers"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Offers) != 2 || reply.Offers[0].ID != "cheap" || reply.Offers[1].ID != "dear" {
		t.Fatalf("offers = %+v", reply.Offers)
	}
	if reply.Offers[0].Status != "unavailable" || reply.Offers[0].Quantity == nil || *reply.Offers[0].Quantity != 0 {
		t.Fatalf("the sold out offer = %+v", reply.Offers[0])
	}
	if reply.Offers[1].Status != "available" || reply.Offers[1].Quantity == nil || *reply.Offers[1].Quantity != 3 {
		t.Fatalf("the stocked offer = %+v", reply.Offers[1])
	}
}

// TestCheckStockRefusesAnOfferOutsideTheSearch Keeps this Route from Becoming a
// Proxy: only an Offer this Search Found can be Asked about.
func TestCheckStockRefusesAnOfferOutsideTheSearch(t *testing.T) {
	if recorder := askStock(t, `{"offers":["https://evil.test/anything"]}`); recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d", recorder.Code)
	}
}

// TestCheckStockRefusesAnEmptyList Answers nothing to ask with a Refusal, not
// with an empty Success that Reads like a finished Check.
func TestCheckStockRefusesAnEmptyList(t *testing.T) {
	if recorder := askStock(t, `{"offers":[]}`); recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
}

// fallenSource Fails the Way a Store Down Fails: it Names itself and Says why.
type fallenSource struct{ name string }

func (f fallenSource) FindOffers(context.Context, offer.CardQuery) ([]offer.Offer, error) {
	return nil, errors.New("source returned HTTP 503")
}
func (f fallenSource) SourceName() string { return f.name }

// TestEverySourceFallingIsAnAnswerNotAFailure Covers the Reply a Caller Gets
// when nothing Answered. The Faults Name who Fell; a 500 Threw them away and
// Left the Caller with a blank Failure it could do nothing about.
func TestEverySourceFallingIsAnAnswerNotAFailure(t *testing.T) {
	store := &fakeStore{}
	service := search.Service{
		Searches: store,
		SourcesByGame: map[search.Game][]search.OfferSource{
			search.GameMagic: {fallenSource{name: "lacripta.cl"}},
		},
	}
	api := API{Searches: service, Health: store, Token: "secret"}
	request := httptest.NewRequest(http.MethodGet,
		"/v1/cards/offers?game=magic&name=Play+Booster&kind=sealed", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()

	api.BuildHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	// El Cuerpo se Mira crudo: `offers` debe Llegar como Arreglo vacío, nunca
	// como `null`. Decodificado, uno y otro se Ven iguales.
	if strings.Contains(response.Body.String(), `"offers":null`) {
		t.Errorf("offers = null, want an empty array: %s", response.Body.String())
	}
	var reply struct {
		Offers []offer.Offer        `json:"offers"`
		Faults []search.SourceFault `json:"faults"`
	}
	if err := json.NewDecoder(response.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Offers) != 0 {
		t.Errorf("offers = %d, want none", len(reply.Offers))
	}
	if len(reply.Faults) != 1 || reply.Faults[0].Source != "lacripta.cl" {
		t.Errorf("faults = %#v", reply.Faults)
	}
}

type fakeShelf struct {
	store, list string
	err         error
}

func (s *fakeShelf) RefreshStoreLists(_ context.Context, store, list string) (int, error) {
	s.store, s.list = store, list
	if s.err != nil {
		return 0, s.err
	}
	return 2, nil
}

func refreshInventory(api API, target string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, target, nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	return response
}

func TestRefreshStoreInventory(t *testing.T) {
	shelf := &fakeShelf{}
	response := refreshInventory(API{Token: "secret", Inventories: shelf}, "/v1/stores/el-wombat-rabioso-tcg/inventory/refresh?list=Negro")

	if response.Code != http.StatusOK || response.Body.String() != `{"store_id":"el-wombat-rabioso-tcg","list":"Negro","refreshed":2}`+"\n" {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if shelf.store != "el-wombat-rabioso-tcg" || shelf.list != "Negro" {
		t.Fatalf("store=%s list=%s", shelf.store, shelf.list)
	}
}

func TestRefreshUnknownStoreAnswersNotFound(t *testing.T) {
	shelf := &fakeShelf{err: moxfield.ErrNoList}
	response := refreshInventory(API{Token: "secret", Inventories: shelf}, "/v1/stores/la-cripta/inventory/refresh")

	if response.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
