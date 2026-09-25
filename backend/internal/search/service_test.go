package search

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type stubProvider struct {
	items []model.Offer
	err   error
}

func (s stubProvider) Search(context.Context, model.CardQuery) ([]model.Offer, error) {
	return s.items, s.err
}

func (s stubProvider) SourceName() string { return "stub" }

type stubCache struct {
	items []model.Offer
	found bool
	saved bool
}

func (c *stubCache) LoadOffers(context.Context, string) ([]model.Offer, bool, error) {
	return c.items, c.found, nil
}

func (c *stubCache) SaveOffers(_ context.Context, _ string, items []model.Offer, _ time.Duration) error {
	c.items, c.saved = items, true
	return nil
}

func TestFindProvider(t *testing.T) {
	wanted := model.Offer{ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10.00", PriceCurrency: "USD"}
	cache := &stubCache{}
	service := Service{Providers: map[model.Game]Provider{model.GameMagic: stubProvider{items: []model.Offer{wanted}}}, Cache: cache}
	items, _, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 || !cache.saved {
		t.Fatalf("FindCardOffers() items=%v saved=%v err=%v", items, cache.saved, err)
	}
}

func TestAnOfferCarriesItsConfiguredLocation(t *testing.T) {
	wanted := model.Offer{
		ID: "one", Store: "Netdecker", Source: "v3.netdecker.cl",
		URL: "https://v3.netdecker.cl/card", PriceAmount: "1000", PriceCurrency: "CLP",
	}
	service := Service{
		Providers:      map[model.Game]Provider{model.GameYuGiOh: stubProvider{items: []model.Offer{wanted}}},
		StoreLocations: map[string][]string{"v3.netdecker.cl": {"Viña del Mar", "Quilpué - Mesa 1"}},
	}

	items, _, err := service.FindCardOffers(context.Background(), model.GameYuGiOh, model.CardQuery{Name: "Kuriboh"})
	locations := []string{"Viña del Mar", "Quilpué - Mesa 1"}
	if err != nil || len(items) != 1 || !slices.Equal(items[0].Locations, locations) {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestFindProviderRejectsUnconfiguredGame(t *testing.T) {
	service := Service{Providers: map[model.Game]Provider{}}
	if _, _, err := service.FindCardOffers(context.Background(), model.GamePokemon, model.CardQuery{Name: "Charizard"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("FindCardOffers() error=%v", err)
	}
}

func TestValidateCreate(t *testing.T) {
	valid := model.CreateInput{Game: model.GameMagic, Cards: []model.CardInput{{Name: "Sol Ring", Quantity: 1}}}
	for _, game := range []model.Game{model.GameMagic, model.GamePokemon, model.GameYuGiOh, model.GameOnePiece} {
		valid.Game = game
		if err := ValidateCreate(valid, 500, 99); err != nil {
			t.Fatalf("ValidateCreate(%q) error = %v", game, err)
		}
	}
	invalidGame := valid
	invalidGame.Game = "unknown"
	if err := ValidateCreate(invalidGame, 500, 99); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateCreate() game error = %v", err)
	}
	valid.Cards[0].Quantity = 0
	if err := ValidateCreate(valid, 500, 99); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateCreate() error = %v", err)
	}
}

type keyRecorder struct{ keys []string }

func (r *keyRecorder) LoadOffers(_ context.Context, key string) ([]model.Offer, bool, error) {
	r.keys = append(r.keys, key)
	return nil, false, nil
}

func (r *keyRecorder) SaveOffers(context.Context, string, []model.Offer, time.Duration) error {
	return nil
}

// TestCacheSeparatesMatchModes Keeps a wide Answer from Serving a narrow Question.
func TestCacheSeparatesMatchModes(t *testing.T) {
	recorder := &keyRecorder{}
	service := Service{
		Providers: map[model.Game]Provider{model.GameMagic: stubProvider{items: []model.Offer{}}},
		Cache:     recorder, CacheNamespace: "ns:",
	}
	for _, mode := range []model.MatchMode{model.MatchExact, model.MatchIncludes} {
		if _, _, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring", Match: mode}); err != nil {
			t.Fatal(err)
		}
	}
	if len(recorder.keys) != 2 || recorder.keys[0] == recorder.keys[1] {
		t.Fatalf("cache keys=%v, want one per mode", recorder.keys)
	}
}

type stubSource struct {
	name  string
	items []model.Offer
	err   error
}

func (s stubSource) FindOffers(context.Context, model.CardQuery) ([]model.Offer, error) {
	return s.items, s.err
}

func (s stubSource) SourceName() string { return s.name }

// TestAPartialAnswerNamesWhatFell Covers the Case that Cost two Hand Probes:
// one Store Falls, another Answers, and the Reply Looks complete.
func TestAPartialAnswerNamesWhatFell(t *testing.T) {
	wanted := model.Offer{ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10", PriceCurrency: "CLP"}
	service := Service{SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {
		stubSource{name: "good.cl", items: []model.Offer{wanted}},
		stubSource{name: "broken.cl", err: errors.New("pagination repeated products")},
	}}}
	items, faults, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	if len(faults) != 1 || faults[0].Source != "broken.cl" || faults[0].Reason == "" {
		t.Fatalf("faults=%+v, want broken.cl named", faults)
	}
}

// TestAFullAnswerNamesNobody Keeps the Field from Crying Wolf.
func TestAFullAnswerNamesNobody(t *testing.T) {
	service := Service{SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {
		stubSource{name: "good.cl", items: []model.Offer{{
			ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10", PriceCurrency: "CLP",
		}}},
	}}}
	_, faults, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(faults) != 0 {
		t.Fatalf("faults=%+v err=%v", faults, err)
	}
}

func TestADirectStoreOverridesItsAggregator(t *testing.T) {
	aggregated := model.Offer{
		ID: "aggregated", Store: "Oasis Games", Source: "scry.cl",
		URL: "https://www.oasisgames.cl/products/sol-ring?variant=10", PriceAmount: "1200", PriceCurrency: "CLP",
	}
	direct := aggregated
	direct.ID, direct.Source, direct.PriceAmount = "direct", "www.oasisgames.cl", "1000"
	service := Service{
		Providers: map[model.Game]Provider{model.GameMagic: stubProvider{items: []model.Offer{aggregated}}},
		SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {
			stubSource{name: "www.oasisgames.cl", items: []model.Offer{direct}},
		}},
	}

	items, _, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 || items[0].ID != "direct" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

func TestAnAggregatorBacksUpAFailedStore(t *testing.T) {
	aggregated := model.Offer{
		ID: "aggregated", Store: "Oasis Games", Source: "scry.cl",
		URL: "https://www.oasisgames.cl/products/sol-ring?variant=10", PriceAmount: "1200", PriceCurrency: "CLP",
	}
	service := Service{
		Providers: map[model.Game]Provider{model.GameMagic: stubProvider{items: []model.Offer{aggregated}}},
		SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {
			stubSource{name: "www.oasisgames.cl", err: errors.New("store unavailable")},
		}},
	}

	items, faults, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 || items[0].ID != "aggregated" || len(faults) != 1 {
		t.Fatalf("items=%+v faults=%+v err=%v", items, faults, err)
	}
}

type stubLibrary struct {
	prints []cardmetadata.Print
	err    error
	calls  int
}

func (l *stubLibrary) CardPrints(context.Context, string) ([]cardmetadata.Print, error) {
	l.calls++
	return l.prints, l.err
}

func storeOffer(id, title, image string) model.Offer {
	return model.Offer{
		ID: id, CardName: title, Store: "store", URL: "https://store.test/" + id,
		PriceAmount: "2800", PriceCurrency: "CLP", Image: image,
		Metadata: map[string]string{"title": title},
	}
}

// TestAStoreOfferBorrowsThePrintImage Covers the Gap the Probes Kept Showing:
// a Store Publishes a Title and a Price, never an Image.
func TestAStoreOfferBorrowsThePrintImage(t *testing.T) {
	library := &stubLibrary{prints: []cardmetadata.Print{
		{Edition: "c21", CollectorNumber: "263", Image: "https://images.test/c21-263.jpg"},
	}}
	service := Service{
		SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {stubSource{name: "store.cl", items: []model.Offer{
			storeOffer("one", "Sol Ring [C21] #263", ""),
			storeOffer("two", "Sol Ring — Near Mint", ""),
			storeOffer("three", "Sol Ring [C21] #263", "https://own.test/picture.jpg"),
		}}}},
		PrintsByGame: map[model.Game]PrintLibrary{model.GameMagic: library},
	}
	items, _, err := service.FindCardOffers(context.Background(), model.GameMagic, model.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 3 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	found := map[string]string{}
	for _, item := range items {
		found[item.ID] = item.Image
	}
	if found["one"] != "https://images.test/c21-263.jpg" {
		t.Errorf("the title named a printing and got %q", found["one"])
	}
	if found["two"] != "" {
		t.Errorf("a title naming no edition invented %q", found["two"])
	}
	if found["three"] != "https://own.test/picture.jpg" {
		t.Errorf("the source's own picture was overwritten with %q", found["three"])
	}
	if library.calls != 1 {
		t.Errorf("asked the print list %d times for one card", library.calls)
	}
}

// TestAGameWithoutAPrintListAsksNobody Keeps Pokemon and Yu-Gi-Oh out of Scryfall.
func TestAGameWithoutAPrintListAsksNobody(t *testing.T) {
	library := &stubLibrary{}
	service := Service{
		SourcesByGame: map[model.Game][]OfferSource{model.GameYuGiOh: {stubSource{name: "store.cl", items: []model.Offer{
			storeOffer("one", "Kuriboh", ""),
		}}}},
		PrintsByGame: map[model.Game]PrintLibrary{model.GameMagic: library},
	}
	if _, _, err := service.FindCardOffers(context.Background(), model.GameYuGiOh, model.CardQuery{Name: "Kuriboh"}); err != nil {
		t.Fatal(err)
	}
	if library.calls != 0 {
		t.Fatalf("asked the Magic print list %d times for Yu-Gi-Oh", library.calls)
	}
}

type stubSets struct {
	names []string
	err   error
	calls int
}

func (s *stubSets) GameSets(context.Context) ([]string, error) {
	s.calls++
	return s.names, s.err
}

// TestASealedAnswerKeepsItsOwnGame Covers what the Probe Showed: a Store
// Selling several Games Answered `Booster Box` with Cardfight!! Vanguard,
// because the Question Named no Game and the Title Named no Card.
func TestASealedAnswerKeepsItsOwnGame(t *testing.T) {
	sets := &stubSets{names: []string{"Chaos Origins", "Rarity Collection 5"}}
	service := Service{
		Providers: map[model.Game]Provider{model.GameYuGiOh: stubProvider{items: []model.Offer{
			{ID: "own", CardName: "Chaos Origins Booster Box Español", Store: "store",
				URL: "https://store.test/own", PriceAmount: "104990", Source: "store"},
			{ID: "other", CardName: "Cardfight!! Vanguard Booster Box: Destined Showdown", Store: "store",
				URL: "https://store.test/other", PriceAmount: "59143", Source: "store"},
		}}},
		SetsByGame: map[model.Game]SetLibrary{model.GameYuGiOh: sets},
	}
	query := model.CardQuery{Name: "Booster Box", Kind: model.KindSealed}
	items, _, err := service.FindCardOffers(context.Background(), model.GameYuGiOh, query)
	if err != nil || len(items) != 1 || items[0].ID != "own" {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

// TestASingleAnswerNeverAsksForSets Keeps the Set List out of the Path it does
// not Serve: a Card Name already Names one Game.
func TestASingleAnswerNeverAsksForSets(t *testing.T) {
	sets := &stubSets{names: []string{"Chaos Origins"}}
	service := Service{
		Providers: map[model.Game]Provider{model.GameYuGiOh: stubProvider{items: []model.Offer{
			{ID: "card", CardName: "Kuriboh", Store: "store", URL: "https://store.test/card",
				PriceAmount: "990", Source: "store"},
		}}},
		SetsByGame: map[model.Game]SetLibrary{model.GameYuGiOh: sets},
	}
	items, _, err := service.FindCardOffers(context.Background(), model.GameYuGiOh, model.CardQuery{Name: "Kuriboh"})
	if err != nil || len(items) != 1 || sets.calls != 0 {
		t.Fatalf("items=%+v calls=%d err=%v", items, sets.calls, err)
	}
}

// TestASealedAnswerSurvivesASetListThatFails Fails open: fewer Offers Hurt a
// Caller more than a Stranger among them.
func TestASealedAnswerSurvivesASetListThatFails(t *testing.T) {
	service := Service{
		Providers: map[model.Game]Provider{model.GameYuGiOh: stubProvider{items: []model.Offer{
			{ID: "own", CardName: "Chaos Origins Booster Box", Store: "store",
				URL: "https://store.test/own", PriceAmount: "104990", Source: "store"},
		}}},
		SetsByGame: map[model.Game]SetLibrary{model.GameYuGiOh: &stubSets{err: errors.New("catalog down")}},
	}
	query := model.CardQuery{Name: "Booster Box", Kind: model.KindSealed}
	items, _, err := service.FindCardOffers(context.Background(), model.GameYuGiOh, query)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
}

// keyCache Remembers which Key each Question Wrote under.
type keyCache struct{ keys []string }

func (c *keyCache) LoadOffers(context.Context, string) ([]model.Offer, bool, error) {
	return nil, false, nil
}

func (c *keyCache) SaveOffers(_ context.Context, key string, _ []model.Offer, _ time.Duration) error {
	c.keys = append(c.keys, key)
	return nil
}

// TestAnOldItemAsksTheQuestionOfANewOne Covers the Items Stored before `kind`
// Existed: they Come back Empty and must Read as Singles, Cache Key included.
func TestAnOldItemAsksTheQuestionOfANewOne(t *testing.T) {
	cache := &keyCache{}
	service := Service{
		Providers: map[model.Game]Provider{model.GameYuGiOh: stubProvider{items: []model.Offer{
			{ID: "card", CardName: "Kuriboh", Store: "store", URL: "https://store.test/card",
				PriceAmount: "990", Source: "store"},
		}}},
		Cache: cache,
	}
	stored := model.CardQuery{Name: "Kuriboh"}
	items, _, err := service.collectOffers(context.Background(), model.GameYuGiOh, stored)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].Kind != model.KindSingle {
		t.Fatalf("an empty kind read as %q", items[0].Kind)
	}
	asked := model.CardQuery{Name: "Kuriboh", Kind: model.KindSingle, Match: model.MatchExact}
	if _, _, err := service.collectOffers(context.Background(), model.GameYuGiOh, asked); err != nil {
		t.Fatal(err)
	}
	if len(cache.keys) != 2 || cache.keys[0] != cache.keys[1] {
		t.Fatalf("a stored item and a fresh question wrote different keys: %q", cache.keys)
	}
}

// TestASealedQuestionSkipsASinglesOnlySource Covers the Mark a Store or an
// Aggregator Carries in its Configuration. A Singles Index Answers a Sealed
// Question with nothing after Spending a Request and its whole Timeout, and
// that Wait Comes back Marked incomplete — which Reads like the Box might be
// somewhere the Search never Looked.
func TestASealedQuestionSkipsASinglesOnlySource(t *testing.T) {
	singles := &countingSource{name: "singles.example.cl"}
	boxes := &countingSource{name: "boxes.example.cl"}
	service := Service{
		SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {singles, boxes}},
		SinglesOnly:   map[string]bool{"singles.example.cl": true},
	}

	if _, _, err := service.FindCardOffers(context.Background(), model.GameMagic,
		model.CardQuery{Name: "Play Booster", Kind: model.KindSealed}); err != nil {
		t.Fatal(err)
	}
	if singles.asked != 0 {
		t.Errorf("a singles-only source was asked for sealed product %d times", singles.asked)
	}
	if boxes.asked != 1 {
		t.Errorf("the sealed source was asked %d times", boxes.asked)
	}

	if _, _, err := service.FindCardOffers(context.Background(), model.GameMagic,
		model.CardQuery{Name: "Sol Ring", Kind: model.KindSingle}); err != nil {
		t.Fatal(err)
	}
	if singles.asked != 1 {
		t.Error("a singles question stopped reaching the singles source")
	}
}

// TestAnUnmarkedSourceStaysAsked Covers the Fail-open Half of the same Mark.
func TestAnUnmarkedSourceStaysAsked(t *testing.T) {
	unmarked := &countingSource{name: "unknown.example.cl"}
	service := Service{SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {unmarked}}}

	if _, _, err := service.FindCardOffers(context.Background(), model.GameMagic,
		model.CardQuery{Name: "Play Booster", Kind: model.KindSealed}); err != nil {
		t.Fatal(err)
	}
	if unmarked.asked != 1 {
		t.Error("an unmarked source was dropped from a sealed question")
	}
}

type countingSource struct {
	name  string
	asked int
}

func (c *countingSource) FindOffers(context.Context, model.CardQuery) ([]model.Offer, error) {
	c.asked++
	return nil, nil
}

func (c *countingSource) SourceName() string { return c.name }

// TestASealedOfferKeepsItsOwnPicture Covers the Image a Box Wears.
//
// The Printing Library Lists Cards, and a Set Name Names both the Set and the
// Cards in it. Filling a Booster Box with the Picture of a Card Printed inside
// it Looks like an Answer and is a Lie: a Box Wears its Store's Photo or none.
func TestASealedOfferKeepsItsOwnPicture(t *testing.T) {
	source := &fixedSource{items: []model.Offer{{
		ID: "1", CardName: "Bloomburrow Play Booster Box", Store: "Tienda",
		PriceAmount: "100000", PriceCurrency: "CLP", URL: "https://example.cl/box",
		Source: "example.cl", StockStatus: "available",
	}}}
	service := Service{
		SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {source}},
		PrintsByGame:  map[model.Game]PrintLibrary{model.GameMagic: stubPrints{}},
	}

	sealed, _, err := service.FindCardOffers(context.Background(), model.GameMagic,
		model.CardQuery{Name: "Bloomburrow", Kind: model.KindSealed})
	if err != nil {
		t.Fatal(err)
	}
	if sealed[0].Image != "" {
		t.Errorf("a box was painted with a card: %q", sealed[0].Image)
	}
}

type fixedSource struct{ items []model.Offer }

func (f *fixedSource) FindOffers(context.Context, model.CardQuery) ([]model.Offer, error) {
	return append([]model.Offer(nil), f.items...), nil
}
func (f *fixedSource) SourceName() string { return "example.cl" }

type stubPrints struct{}

func (stubPrints) CardPrints(context.Context, string) ([]cardmetadata.Print, error) {
	return []cardmetadata.Print{{
		Edition: "blb", EditionName: "Bloomburrow", CollectorNumber: "1",
		Image: "https://img/card.jpg",
	}}, nil
}
