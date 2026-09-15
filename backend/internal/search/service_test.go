package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type stubProvider struct {
	items []offer.Offer
	err   error
}

func (s stubProvider) Search(context.Context, offer.CardQuery) ([]offer.Offer, error) {
	return s.items, s.err
}

func (s stubProvider) SourceName() string { return "stub" }

type stubCache struct {
	items []offer.Offer
	found bool
	saved bool
}

func (c *stubCache) LoadOffers(context.Context, string) ([]offer.Offer, bool, error) {
	return c.items, c.found, nil
}

func (c *stubCache) SaveOffers(_ context.Context, _ string, items []offer.Offer, _ time.Duration) error {
	c.items, c.saved = items, true
	return nil
}

func TestFindProvider(t *testing.T) {
	wanted := offer.Offer{ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10.00", PriceCurrency: "USD"}
	cache := &stubCache{}
	service := Service{Providers: map[Game]Provider{GameMagic: stubProvider{items: []offer.Offer{wanted}}}, Cache: cache}
	items, _, err := service.FindCardOffers(context.Background(), GameMagic, offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 || !cache.saved {
		t.Fatalf("FindCardOffers() items=%v saved=%v err=%v", items, cache.saved, err)
	}
}

func TestFindProviderRejectsUnconfiguredGame(t *testing.T) {
	service := Service{Providers: map[Game]Provider{}}
	if _, _, err := service.FindCardOffers(context.Background(), GamePokemon, offer.CardQuery{Name: "Charizard"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("FindCardOffers() error=%v", err)
	}
}

func TestValidateCreate(t *testing.T) {
	valid := CreateInput{Game: GameMagic, Cards: []CardInput{{Name: "Sol Ring", Quantity: 1}}}
	for _, game := range []Game{GameMagic, GamePokemon, GameYuGiOh, GameOnePiece} {
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

func (r *keyRecorder) LoadOffers(_ context.Context, key string) ([]offer.Offer, bool, error) {
	r.keys = append(r.keys, key)
	return nil, false, nil
}

func (r *keyRecorder) SaveOffers(context.Context, string, []offer.Offer, time.Duration) error {
	return nil
}

// TestCacheSeparatesMatchModes Keeps a wide Answer from Serving a narrow Question.
func TestCacheSeparatesMatchModes(t *testing.T) {
	recorder := &keyRecorder{}
	service := Service{
		Providers: map[Game]Provider{GameMagic: stubProvider{items: []offer.Offer{}}},
		Cache:     recorder, CacheNamespace: "ns:",
	}
	for _, mode := range []offer.MatchMode{offer.MatchExact, offer.MatchIncludes} {
		if _, _, err := service.FindCardOffers(context.Background(), GameMagic, offer.CardQuery{Name: "Sol Ring", Match: mode}); err != nil {
			t.Fatal(err)
		}
	}
	if len(recorder.keys) != 2 || recorder.keys[0] == recorder.keys[1] {
		t.Fatalf("cache keys=%v, want one per mode", recorder.keys)
	}
}

type stubSource struct {
	name  string
	items []offer.Offer
	err   error
}

func (s stubSource) FindOffers(context.Context, offer.CardQuery) ([]offer.Offer, error) {
	return s.items, s.err
}

func (s stubSource) SourceName() string { return s.name }

// TestAPartialAnswerNamesWhatFell Covers the Case that Cost two Hand Probes:
// one Store Falls, another Answers, and the Reply Looks complete.
func TestAPartialAnswerNamesWhatFell(t *testing.T) {
	wanted := offer.Offer{ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10", PriceCurrency: "CLP"}
	service := Service{SourcesByGame: map[Game][]OfferSource{GameMagic: {
		stubSource{name: "good.cl", items: []offer.Offer{wanted}},
		stubSource{name: "broken.cl", err: errors.New("pagination repeated products")},
	}}}
	items, faults, err := service.FindCardOffers(context.Background(), GameMagic, offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	if len(faults) != 1 || faults[0].Source != "broken.cl" || faults[0].Reason == "" {
		t.Fatalf("faults=%+v, want broken.cl named", faults)
	}
}

// TestAFullAnswerNamesNobody Keeps the Field from Crying Wolf.
func TestAFullAnswerNamesNobody(t *testing.T) {
	service := Service{SourcesByGame: map[Game][]OfferSource{GameMagic: {
		stubSource{name: "good.cl", items: []offer.Offer{{
			ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10", PriceCurrency: "CLP",
		}}},
	}}}
	_, faults, err := service.FindCardOffers(context.Background(), GameMagic, offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(faults) != 0 {
		t.Fatalf("faults=%+v err=%v", faults, err)
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

func storeOffer(id, title, image string) offer.Offer {
	return offer.Offer{
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
		SourcesByGame: map[Game][]OfferSource{GameMagic: {stubSource{name: "store.cl", items: []offer.Offer{
			storeOffer("one", "Sol Ring [C21] #263", ""),
			storeOffer("two", "Sol Ring — Near Mint", ""),
			storeOffer("three", "Sol Ring [C21] #263", "https://own.test/picture.jpg"),
		}}}},
		PrintsByGame: map[Game]PrintLibrary{GameMagic: library},
	}
	items, _, err := service.FindCardOffers(context.Background(), GameMagic, offer.CardQuery{Name: "Sol Ring"})
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
		SourcesByGame: map[Game][]OfferSource{GameYuGiOh: {stubSource{name: "store.cl", items: []offer.Offer{
			storeOffer("one", "Kuriboh", ""),
		}}}},
		PrintsByGame: map[Game]PrintLibrary{GameMagic: library},
	}
	if _, _, err := service.FindCardOffers(context.Background(), GameYuGiOh, offer.CardQuery{Name: "Kuriboh"}); err != nil {
		t.Fatal(err)
	}
	if library.calls != 0 {
		t.Fatalf("asked the Magic print list %d times for Yu-Gi-Oh", library.calls)
	}
}
