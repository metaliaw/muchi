package search

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type stubSource struct {
	items []offer.Offer
	err   error
}

func (s stubSource) FindOffers(context.Context, string) ([]offer.Offer, error) {
	return s.items, s.err
}

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

func TestFindFallback(t *testing.T) {
	wanted := offer.Offer{ID: "one", Store: "store", URL: "https://store.test", PriceAmount: "10.00", PriceCurrency: "USD"}
	cache := &stubCache{}
	service := Service{Sources: []OfferSource{stubSource{err: errors.New("down")}, stubSource{items: []offer.Offer{wanted}}}, Cache: cache}
	items, err := service.FindCardOffers(context.Background(), "Sol Ring")
	if err != nil || len(items) != 1 || !cache.saved {
		t.Fatalf("FindCardOffers() items=%v saved=%v err=%v", items, cache.saved, err)
	}
}

func TestValidateCreate(t *testing.T) {
	valid := CreateInput{Cards: []CardInput{{Name: "Sol Ring", Quantity: 1}}}
	if err := ValidateCreate(valid, 500, 99); err != nil {
		t.Fatalf("ValidateCreate() error = %v", err)
	}
	valid.Cards[0].Quantity = 0
	if err := ValidateCreate(valid, 500, 99); !errors.Is(err, ErrInvalid) {
		t.Fatalf("ValidateCreate() error = %v", err)
	}
}
