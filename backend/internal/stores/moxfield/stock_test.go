package moxfield

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// findOffer Takes one Offer of the Fixture the same Way a Search would.
func findOffer(t *testing.T, client *Client, name string) offer.Offer {
	t.Helper()
	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: name})
	if err != nil || len(items) == 0 {
		t.Fatalf("offers=%v err=%v", items, err)
	}
	return items[0]
}

func TestTheListAnswersItsOwnStock(t *testing.T) {
	client, fetcher, _ := newTestClient(t)
	item := findOffer(t, client, "Caged Sun")
	shelf := Shelf{Lists: []*Client{client}}

	reading, err := shelf.CheckStock(context.Background(), item)

	if err != nil || reading.Status != "available" || reading.Quantity == nil || *reading.Quantity < 1 {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
	// Two Visits: the Search Read the List, and the Check Read it again. A
	// Check Served from the Cache would Confirm the same Inventory it Doubts.
	if fetcher.calls != 2 {
		t.Fatalf("calls=%d", fetcher.calls)
	}
}

func TestAnEntryGoneFromTheListIsSold(t *testing.T) {
	client, _, _ := newTestClient(t)
	shelf := Shelf{Lists: []*Client{client}}

	reading, err := shelf.CheckStock(context.Background(), offer.Offer{
		ID: "moxfield:wombat:yqdRPdoUlEiFz21qKYHGMA:ido", Source: "moxfield",
	})

	if err != nil || reading.Status != "unavailable" {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
}

func TestAnOfferOfAnotherShelfStaysUnknown(t *testing.T) {
	client, _, _ := newTestClient(t)

	reading, err := (Shelf{Lists: []*Client{client}}).CheckStock(
		context.Background(), offer.Offer{ID: "moxfield:otra:lista:x", Source: "moxfield"})

	if err != ErrNoList || reading.Status != "unknown" {
		t.Fatalf("reading=%+v err=%v", reading, err)
	}
}
