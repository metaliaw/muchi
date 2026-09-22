package stores

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type listAdapter struct{ asked offer.Offer }

func (a *listAdapter) CheckStock(_ context.Context, item offer.Offer) (offer.StockReading, error) {
	a.asked = item
	return offer.CountStock("unavailable", 0), nil
}

// A Store that Publishes Lists Points every Offer at the same Deck URL. Asking
// the Host would Name no Store and Leave the Answer "unknown" forever.
func TestAListedOfferAsksItsShelf(t *testing.T) {
	shelf := &listAdapter{}
	checker := Checker{Config: Config{}, Lists: shelf}

	reading, err := checker.CheckStock(context.Background(), offer.Offer{
		ID: "moxfield:wombat:lista:x", Source: "moxfield",
		URL: "https://moxfield.com/decks/yqdRPdoUlEiFz21qKYHGMA",
	})

	if err != nil || reading.Status != "unavailable" {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
	if shelf.asked.ID != "moxfield:wombat:lista:x" {
		t.Fatalf("asked=%q", shelf.asked.ID)
	}
}
