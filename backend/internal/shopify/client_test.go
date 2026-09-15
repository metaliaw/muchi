package shopify

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type fixtureFetcher func(string) ([]byte, error)

func (f fixtureFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	return f(target)
}

const productFixture = `{"id":1,"title":"Sol Ring (0356) [FIC-356]","options":[{"name":"Condition","position":1},{"name":"Language","position":2}],"variants":[{"id":10,"title":"NM / English","price":350000,"available":true,"options":["NM","English"]},{"id":11,"price":400000,"available":false},{"id":12,"price":450000,"available":true}]}`

func TestFindOffers(t *testing.T) {
	calls := map[string]int{}
	client := Client{Domain: "cards.test", Name: "Cards", Fetcher: fixtureFetcher(func(target string) ([]byte, error) {
		u, _ := url.Parse(target)
		calls[u.Path]++
		switch u.Path {
		case "/search":
			if u.Query().Get("q") != `"Sol Ring"` || u.Query().Get("filter.v.availability") != "1" {
				t.Fatalf("query=%s", target)
			}
			if u.Query().Get("page") == "1" {
				return []byte(`<input name="q"><a href="/products/ring">Ring</a><a href="https://evil.test/products/other">Other</a><a href="/search?page=2">Next</a>`), nil
			}
			return []byte(`<input name="q"><a href="/products/ring">Duplicate</a><a href="/collections/cards/products/token">Token</a>`), nil
		case "/cart.js":
			return []byte(`{"currency":"CLP"}`), nil
		case "/products/ring.js":
			return []byte(productFixture), nil
		case "/products/token.js":
			return []byte(`{"id":2,"title":"Sol Ring Token","variants":[]}`), nil
		}
		return nil, errors.New("unexpected request: " + target)
	})}
	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%+v err=%v", items, err)
	}
	if items[0].PriceAmount != "3500" || items[0].PriceCurrency != "CLP" || items[0].Condition != "NM" || items[0].Language != "English" || items[0].URL == items[1].URL || calls["/products/ring.js"] != 1 {
		t.Fatalf("items=%+v calls=%v", items, calls)
	}
	for _, test := range []struct{ id, status string }{{"10", "available"}, {"11", "unavailable"}, {"99", "unknown"}, {"", "unknown"}} {
		status, err := client.CheckStock(context.Background(), offer.Offer{URL: "https://cards.test/products/ring?variant=" + test.id})
		if err != nil || status != test.status {
			t.Fatalf("id=%s status=%s err=%v", test.id, status, err)
		}
	}
}

func TestInvalidResponses(t *testing.T) {
	for _, test := range []struct{ name, search, cart, product string }{
		{"challenge", `<html>Challenge</html>`, "", ""},
		{"currency", `<input name="q"><a href="/products/ring">Ring</a>`, `{}`, productFixture},
		{"product", `<input name="q"><a href="/products/ring">Ring</a>`, `{"currency":"CLP"}`, `{}`},
		{"price", `<input name="q"><a href="/products/ring">Ring</a>`, `{"currency":"CLP"}`, `{"id":1,"title":"Sol Ring","variants":[{"id":1,"available":true}]}`},
		{"pagination", `<input name="q"><a href="/products/ring">Ring</a><a href="/search?page=2">Next</a>`, "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := Client{Domain: "cards.test", Fetcher: fixtureFetcher(func(target string) ([]byte, error) {
				u, _ := url.Parse(target)
				switch u.Path {
				case "/search":
					return []byte(test.search), nil
				case "/cart.js":
					return []byte(test.cart), nil
				default:
					return []byte(test.product), nil
				}
			})}
			if _, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"}); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	failure := errors.New("HTTP 429")
	client := Client{Domain: "cards.test", Fetcher: fixtureFetcher(func(string) ([]byte, error) { return nil, failure })}
	if _, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"}); !errors.Is(err, failure) {
		t.Fatalf("err=%v", err)
	}
}

func TestMatchCard(t *testing.T) {
	for _, title := range []string{"Sol Ring", "Sol Ring (2683) [Secret Lair Drop Series]", "Sol Ring - 212 - uncommon", "Sol Ring (0356) [FIC-356]"} {
		if !offer.MatchesCard(title, "Sol Ring") {
			t.Errorf("rejected %s", title)
		}
	}
	for _, title := range []string{"Sol Ring Token", "Sol Ringlet", "Other Sol Ring"} {
		if offer.MatchesCard(title, "Sol Ring") {
			t.Errorf("accepted %s", title)
		}
	}
	if formatPrice(1299) != "12.99" || formatPrice(1) != "0.01" {
		t.Fatal("fractional price")
	}
}

// TestOfferKeepsTheStoreTitle Covers a Store Title that Differs from the Search.
// Naming the Offer after the Question Hides which Printing it really is, and
// leaves every Offer of a wide Search Wearing the same Name.
func TestOfferKeepsTheStoreTitle(t *testing.T) {
	client := Client{Domain: "cards.test", Fetcher: fixtureFetcher(func(target string) ([]byte, error) {
		u, _ := url.Parse(target)
		switch u.Path {
		case "/search":
			return []byte(`<input name="q"><a href="/products/ring">Ring</a>`), nil
		case "/cart.js":
			return []byte(`{"currency":"CLP"}`), nil
		case "/products/ring.js":
			return []byte(productFixture), nil
		}
		return nil, errors.New("unexpected request: " + target)
	})}
	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) == 0 {
		t.Fatalf("items=%d err=%v", len(items), err)
	}
	for _, item := range items {
		if item.CardName != "Sol Ring (0356) [FIC-356]" {
			t.Fatalf("CardName = %q, want the store title", item.CardName)
		}
	}
}
