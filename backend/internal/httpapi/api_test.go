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
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
)

type fakeStore struct {
	job  model.Job
	page model.ResultPage
}

type fakeMetadata struct{}

func (fakeMetadata) CardMetadata(_ context.Context, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	return cardmetadata.Metadata{Name: request.Name, Image: "https://images.example/pikachu.jpg"}, nil
}

func (fakeMetadata) Autocomplete(_ context.Context, name, _ string) ([]string, error) {
	return []string{name, name + " VMAX"}, nil
}

func (s *fakeStore) CreateSearch(_ context.Context, _, _ string, input model.CreateInput) (model.Job, error) {
	s.job = model.Job{ID: "search_one", Status: model.JobQueued, Total: len(input.Cards), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	return s.job, nil
}
func (s *fakeStore) GetSearch(context.Context, string) (model.Job, error) { return s.job, nil }
func (s *fakeStore) ListResults(_ context.Context, _ string, page model.ResultPage) (model.Result, error) {
	s.page = page
	return model.Result{SearchID: s.job.ID, Items: []model.Item{}, Cursor: page.After}, nil
}
func (s *fakeStore) CancelSearch(context.Context, string, string, string) (model.Job, error) {
	s.job.Status = model.JobCancelled
	return s.job, nil
}
func (*fakeStore) ClaimSearchItem(context.Context, string, time.Duration) (model.Item, error) {
	return model.Item{}, search.ErrNotFound
}
func (*fakeStore) RenewItemLease(context.Context, string, string, time.Duration) error { return nil }
func (*fakeStore) CompleteSearchItem(context.Context, model.Item, []model.Offer) error { return nil }
func (*fakeStore) CheckHealth(context.Context) error                                   { return nil }
func (*fakeStore) ListSourceHealth(context.Context) ([]model.SourceHealth, error)      { return nil, nil }

func TestCreateSearch(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Repository: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
	body := `{"game":"magic","cards":[{"name":"Sol Ring","quantity":1}],"options":{"verify_stock":true}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Idempotency-Key", "request-one")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("POST /v1/searches status = %d body=%s", response.Code, response.Body.String())
	}
	var job model.Job
	if err := json.NewDecoder(response.Body).Decode(&job); err != nil || job.ID != "search_one" {
		t.Fatalf("POST /v1/searches job=%#v err=%v", job, err)
	}
}

func TestDecodeSearchDefaultsToMagic(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"cards":[{"name":"Sol Ring","quantity":1}]}`))

	input, err := decodeSearch(request)

	if err != nil || input.Game != model.GameMagic {
		t.Fatalf("decodeSearch() game=%q err=%v", input.Game, err)
	}
}

func TestDecodeSearchKeepsExplicitGame(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"game":"pokemon","cards":[{"name":"Pikachu","quantity":1}]}`))

	input, err := decodeSearch(request)

	if err != nil || input.Game != model.GamePokemon {
		t.Fatalf("decodeSearch() game=%q err=%v", input.Game, err)
	}
}

func TestDecodeSearchPromotesHTTPBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"game":"magic","cards":[{"name":"Sol Ring","quantity":2}],"options":{"verify_stock":true,"match":"includes","kind":"single"}}`))

	input, err := decodeSearch(request)

	if err != nil || input.Game != model.GameMagic || len(input.Cards) != 1 ||
		input.Cards[0].Name != "Sol Ring" || input.Cards[0].Quantity != 2 ||
		!input.Options.VerifyStock || input.Options.Match != model.MatchIncludes ||
		input.Options.Kind != model.KindSingle {
		t.Fatalf("decoded input=%+v err=%v", input, err)
	}
}

func TestDecodeSearchRejectsUnknownHTTPMode(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"cards":[{"name":"Sol Ring","quantity":1}],"options":{"match":"approximate"}}`))

	if _, err := decodeSearch(request); !errors.Is(err, search.ErrInvalid) {
		t.Fatalf("decodeSearch() error=%v", err)
	}
}

func TestRequireHeaders(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Repository: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
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
		CardMetadata: map[model.Game]cardmetadata.Provider{model.GamePokemon: fakeMetadata{}},
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
		Autocomplete: map[model.Game]cardmetadata.AutocompleteProvider{model.GamePokemon: fakeMetadata{}},
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

func TestListResultPage(t *testing.T) {
	store := &fakeStore{job: model.Job{ID: "search_one"}}
	api := API{Searches: search.Service{Repository: store}, Token: "secret"}
	request := httptest.NewRequest(http.MethodGet, "/v1/searches/search_one/results?after=12&limit=25", nil)
	request.Header.Set("Authorization", "Bearer secret")
	response := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(response, request)
	if response.Code != http.StatusOK || store.page != (model.ResultPage{After: 12, Limit: 25}) {
		t.Fatalf("result page=%#v status=%d body=%s", store.page, response.Code, response.Body.String())
	}
}

func TestRejectResultPage(t *testing.T) {
	store := &fakeStore{}
	api := API{Searches: search.Service{Repository: store}, Token: "secret"}
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
	api := API{Searches: search.Service{Repository: store, MaxCards: 500, MaxQuantity: 99}, Health: store, Token: "secret"}
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

func (s *stockStore) ListResults(context.Context, string, model.ResultPage) (model.Result, error) {
	return model.Result{SearchID: "search_one", Items: []model.Item{{ID: "item-1", Offers: []model.Offer{
		{ID: "cheap", URL: "https://store.test/cheap", PriceAmount: "1000"},
		{ID: "dear", URL: "https://store.test/dear", PriceAmount: "2000"},
	}}}}, nil
}

// countingStock Answers Sold Out for the cheap Offer and counts its Visits.
type countingStock struct{ visits []string }

func (c *countingStock) CheckStock(_ context.Context, item model.Offer) (model.StockReading, error) {
	c.visits = append(c.visits, item.ID)
	if item.ID == "cheap" {
		return model.CountStock("unavailable", 0), nil
	}
	return model.CountStock("available", 3), nil
}

func askStock(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	store := &stockStore{}
	api := API{Searches: search.Service{Repository: store, Stocks: &countingStock{}}, Token: "secret"}
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
		Offers []offerStockDTO `json:"offers"`
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

func (f fallenSource) FindOffers(context.Context, model.CardQuery) ([]model.Offer, error) {
	return nil, errors.New("source returned HTTP 503")
}
func (f fallenSource) SourceName() string { return f.name }

// TestEverySourceFallingIsAnAnswerNotAFailure Covers the Reply a Caller Gets
// when nothing Answered. The Faults Name who Fell; a 500 Threw them away and
// Left the Caller with a blank Failure it could do nothing about.
func TestEverySourceFallingIsAnAnswerNotAFailure(t *testing.T) {
	store := &fakeStore{}
	service := search.Service{
		Repository: store,
		SourcesByGame: map[model.Game][]search.OfferSource{
			model.GameMagic: {fallenSource{name: "lacripta.cl"}},
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
		Offers []offerDTO       `json:"offers"`
		Faults []sourceFaultDTO `json:"faults"`
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

// checkoutStore Answers Offers from a Shopify Store, a Store without a Cart
// Link, and a Second Unit of the same Shopify Variant.
type checkoutStore struct{ fakeStore }

func (s *checkoutStore) ListResults(context.Context, string, model.ResultPage) (model.Result, error) {
	return model.Result{SearchID: "search_one", Items: []model.Item{{ID: "item-1", Offers: []model.Offer{
		{ID: "ring", Store: "Shop", URL: "https://shop.test/products/sol-ring?variant=11", Source: "shop.test"},
		{ID: "bolt", Store: "Shop", URL: "https://shop.test/products/bolt", VariantID: "22", Source: "shop.test"},
		{ID: "woo", Store: "Woo", URL: "https://woo.test/producto/sol-ring", VariantID: "7", Source: "woo.test"},
	}}}}, nil
}

func askCheckout(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	links := stores.Config{Stores: map[string]stores.StoreConfig{
		"shop.test": {Platform: "shopify", Enabled: true},
		"woo.test":  {Platform: "woocommerce", Enabled: true},
	}}
	api := API{Searches: search.Service{Repository: &checkoutStore{}, Checkouts: links}, Token: "secret"}
	request := httptest.NewRequest(http.MethodPost, "/v1/searches/search_one/checkout", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	recorder := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(recorder, request)
	return recorder
}

// TestCheckoutLinksGroupsLinesByStore Spells the Contract: one Entry per Store
// in the Order first Asked, a Cart Link where the Platform has one, and the
// Product Pages everywhere else. A single WooCommerce Line gets its own
// `add-to-cart` Link; a second one would not.
func TestCheckoutLinksGroupsLinesByStore(t *testing.T) {
	recorder := askCheckout(t, `{"items":[{"offer_id":"ring","quantity":2},{"offer_id":"woo","quantity":1},{"offer_id":"bolt","quantity":4}]}`)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body)
	}
	var reply struct {
		Stores []storeCheckoutDTO `json:"stores"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if len(reply.Stores) != 2 {
		t.Fatalf("stores = %+v", reply.Stores)
	}
	shop, woo := reply.Stores[0], reply.Stores[1]
	if shop.Domain != "shop.test" || shop.Mode != "cart" || shop.URL != "https://shop.test/cart/11:2,22:4" || len(shop.Lines) != 2 {
		t.Fatalf("shop = %+v", shop)
	}
	if woo.Mode != "cart" || woo.URL != "https://woo.test/?add-to-cart=7&quantity=1" || woo.Lines[0].URL != "https://woo.test/producto/sol-ring" {
		t.Fatalf("woo = %+v", woo)
	}
}

// TestCheckoutLinksRefusesUnknownOffersAndBadQuantities Keeps the Route from
// Linking a URL of the Caller's Choosing or an Absurd Cart.
func TestCheckoutLinksRefusesUnknownOffersAndBadQuantities(t *testing.T) {
	if recorder := askCheckout(t, `{"items":[{"offer_id":"https://evil.test/x","quantity":1}]}`); recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown offer status = %d", recorder.Code)
	}
	for _, body := range []string{`{"items":[]}`, `{"items":[{"offer_id":"ring","quantity":0}]}`, `{"items":[{"offer_id":"ring","quantity":100}]}`} {
		if recorder := askCheckout(t, body); recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s status = %d", body, recorder.Code)
		}
	}
}

// fixedQuoter Prices every Store but the Shopify one, which it cannot Ask.
type fixedQuoter struct{ address model.ShippingAddress }

func (q *fixedQuoter) QuoteCart(_ context.Context, domain string, lines []model.CartLine, address model.ShippingAddress) (model.CartQuote, error) {
	q.address = address
	if domain != "woo.test" {
		return model.CartQuote{}, model.ErrNoQuote
	}
	return model.CartQuote{Currency: "CLP", Total: "7990", Lines: []model.QuoteLine{{OfferID: lines[0].Offer.ID, Quantity: lines[0].Quantity}}}, nil
}

// TestCheckoutLinksQuotesStoresWhenAskedWithAnAddress Keeps the Quote Optional:
// only an Address Asks the Stores, and a Store that cannot be Asked Stays Silent.
func TestCheckoutLinksQuotesStoresWhenAskedWithAnAddress(t *testing.T) {
	quoter := &fixedQuoter{}
	api := API{Searches: search.Service{Repository: &checkoutStore{}, Quotes: quoter}, Token: "secret"}
	body := `{"items":[{"offer_id":"ring","quantity":1},{"offer_id":"woo","quantity":2}],"shipping":{"country":"CL","region":"CL-RM","city":"Santiago"}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/searches/search_one/checkout", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer secret")
	recorder := httptest.NewRecorder()
	api.BuildHandler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", recorder.Code, recorder.Body)
	}
	var reply struct {
		Stores []storeCheckoutDTO `json:"stores"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Stores[0].Quote != nil || reply.Stores[0].QuoteError != "" {
		t.Fatalf("shop = %+v", reply.Stores[0])
	}
	if quote := reply.Stores[1].Quote; quote == nil || quote.Total != "7990" || quote.Lines[0].Quantity != 2 {
		t.Fatalf("woo = %+v", reply.Stores[1])
	}
	if quoter.address.Region != "CL-RM" {
		t.Fatalf("address = %+v", quoter.address)
	}
	if recorder := askCheckout(t, `{"items":[{"offer_id":"woo","quantity":1}],"shipping":{"city":"Santiago"}}`); recorder.Code != http.StatusBadRequest {
		t.Fatalf("address without country status = %d", recorder.Code)
	}
}
