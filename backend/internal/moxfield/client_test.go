package moxfield

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type inventoryFetcher struct {
	data           []byte
	calls          int
	domain, target string
	err            error
}

func (f *inventoryFetcher) FetchSource(_ context.Context, domain, target string) ([]byte, error) {
	f.calls++
	f.domain, f.target = domain, target
	return f.data, f.err
}

type inventoryCache struct {
	items []offer.Offer
	found bool
	key   string
	ttl   time.Duration
}

func (c *inventoryCache) LoadOffers(_ context.Context, key string) ([]offer.Offer, bool, error) {
	return c.items, c.found && c.key == key, nil
}
func (c *inventoryCache) SaveOffers(_ context.Context, key string, items []offer.Offer, ttl time.Duration) error {
	c.items, c.found, c.key, c.ttl = items, true, key, ttl
	return nil
}
func newTestClient(t *testing.T) (*Client, *inventoryFetcher, *inventoryCache) {
	t.Helper()
	data, err := os.ReadFile("testdata/inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	fetcher := &inventoryFetcher{data: data}
	cache := &inventoryCache{}
	return &Client{Fetcher: fetcher, Cache: cache, TTL: 15 * time.Minute, StoreID: "wombat", Store: "El Wombat", ListURL: "https://moxfield.com/decks/yqdRPdoUlEiFz21qKYHGMA", Label: "Artefacto", Rate: 700}, fetcher, cache
}
func TestDynamicInventory(t *testing.T) {
	client, fetcher, cache := newTestClient(t)
	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"})
	if err != nil || len(items) != 3 {
		t.Fatalf("offers=%v err=%v", items, err)
	}
	prices := map[string]string{"nph": "3843", "plst": "2793", "brr": "3143"}
	copies := map[string]string{"nph": "1", "plst": "3", "brr": "1"}
	for _, item := range items {
		if item.PriceAmount != prices[item.Metadata["set"]] || item.Metadata["quantity"] != copies[item.Metadata["set"]] || item.Source != "moxfield" || item.StockStatus != "unknown" {
			t.Fatalf("offer=%+v", item)
		}
	}
	if fetcher.domain != "api2.moxfield.com" || !strings.Contains(fetcher.target, "/v3/decks/all/") {
		t.Fatalf("target=%s", fetcher.target)
	}
	// A Different Card Reuses the Full List, not a Cached Single-Card Result.
	if items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Missing"}); err != nil || len(items) != 0 {
		t.Fatalf("missing=%v err=%v", items, err)
	}
	if fetcher.calls != 1 || cache.ttl != 15*time.Minute {
		t.Fatalf("calls=%d ttl=%v", fetcher.calls, cache.ttl)
	}
	cache.found = false
	fetcher.data = []byte(`{"boards":{"mainboard":{"cards":{}}}}`)
	items, err = client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"})
	if err != nil || len(items) != 0 || fetcher.calls != 2 {
		t.Fatalf("refresh=%v calls=%d err=%v", items, fetcher.calls, err)
	}
}
func TestInventoryFailures(t *testing.T) {
	client, fetcher, cache := newTestClient(t)
	fetcher.err = source.StatusError{Code: 403}
	_, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"})
	var status source.StatusError
	if !errors.As(err, &status) || status.Code != 403 || cache.found {
		t.Fatalf("err=%v cache=%v", err, cache.found)
	}
	fetcher.err = nil
	fetcher.data = []byte(`<html>Blocked</html>`)
	if _, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"}); err == nil || cache.found {
		t.Fatal("HTML cached as empty inventory")
	}
	fetcher.data = []byte(`{}`)
	if _, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"}); err == nil {
		t.Fatal("missing boards accepted")
	}
}
func TestListURLs(t *testing.T) {
	for _, value := range []string{"https://moxfield.com/decks/abc_-/?view=table", "https://www.moxfield.com/decks/abc_-"} {
		id, err := ExtractListID(value)
		if err != nil || id != "abc_-" {
			t.Fatalf("id=%s err=%v", id, err)
		}
	}
	for _, value := range []string{"https://evil.test/decks/abc", "https://moxfield.com.evil.test/decks/abc", "https://moxfield.com/decks/", "https://moxfield.com/decks/a/b", "https://user@moxfield.com/decks/abc"} {
		if _, err := ExtractListID(value); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
func TestFinishPrices(t *testing.T) {
	client, _, _ := newTestClient(t)
	data := []byte(`{"boards":{"mainboard":{"cards":{
 "normal":{"quantity":1,"card":{"id":"a","name":"Card","prices":{"ck":1.99,"ck_foil":8}}},
 "foil":{"quantity":2,"isFoil":true,"card":{"id":"a","name":"Card","prices":{"ck":1.99,"ck_foil":"8.49"}}},
 "etched":{"quantity":1,"finish":"etched","card":{"id":"a","name":"Card","prices":{"ck_etched":9.99}}},
 "unpriced":{"quantity":1,"finish":"foil","card":{"id":"b","name":"Card","prices":{"ck":99}}},
 "empty":{"quantity":0,"card":{"id":"c","name":"Card","prices":{"ck":99}}}
 }}}}`)
	items, missing, err := client.readInventory(data, "abc")
	if err != nil || missing != 1 || len(items) != 3 {
		t.Fatalf("offers=%v missing=%d err=%v", items, missing, err)
	}
	want := map[string]string{"nonfoil": "1393", "foil": "5943", "etched": "6993"}
	for _, item := range items {
		if item.PriceAmount != want[item.Finish] {
			t.Fatalf("offer=%+v", item)
		}
	}
}
func TestPriceConversion(t *testing.T) {
	for _, test := range []struct {
		price string
		rate  int
		want  string
	}{{`"1.99"`, 700, "1393"}, {`1.999`, 500, "1000"}, {`null`, 700, ""}, {`0`, 700, ""}, {`-1`, 700, ""}, {`"bad"`, 700, ""}} {
		got, ok := convertPrice([]byte(test.price), test.rate)
		if got != test.want || ok != (test.want != "") {
			t.Fatalf("price=%s got=%s ok=%v", test.price, got, ok)
		}
	}
}

func TestConcurrentInventory(t *testing.T) {
	client, fetcher, _ := newTestClient(t)
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Go(func() {
			items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Caged Sun"})
			if err != nil || len(items) != 3 {
				t.Errorf("offers=%d err=%v", len(items), err)
			}
		})
	}
	group.Wait()
	if fetcher.calls != 1 {
		t.Fatalf("concurrent requests fetched %d times", fetcher.calls)
	}
}
