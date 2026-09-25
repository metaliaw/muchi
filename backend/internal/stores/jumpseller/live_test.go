package jumpseller_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/stores/jumpseller"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

func TestLiveSolRing(t *testing.T) {
	if os.Getenv("MUCHI_TEST_JUMPSELLER_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_JUMPSELLER_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	for domain, store := range config.Stores {
		if store.Enabled && store.Platform == "jumpseller" {
			t.Run(domain, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
				defer cancel()
				client := jumpseller.Client{Domain: domain, Name: store.Name, Fetcher: &loggedFetcher{test: t, client: source.Client{HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2, BaseDelay: time.Second, UserAgent: "muchi-api/1.0"}}}
				items, err := client.FindOffers(ctx, model.CardQuery{Name: "Sol Ring"})
				if err != nil || len(items) == 0 {
					t.Fatalf("offers=%d err=%v", len(items), err)
				}
				for _, item := range items {
					if item.PriceCurrency != "CLP" || item.StockStatus != "available" {
						t.Fatalf("offer=%+v", item)
					}
				}
				reading, err := client.CheckStock(ctx, items[0])
				if err != nil || reading.Status != "available" {
					t.Fatalf("stock=%s err=%v", reading.Status, err)
				}
				t.Logf("Sol Ring: %d Offers; First Price: %s CLP; Variant Stock: %s", len(items), items[0].PriceAmount, reading.Status)
			})
		}
	}
}

func TestLiveDecksCardsDarkMagician(t *testing.T) {
	if os.Getenv("MUCHI_TEST_JUMPSELLER_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_JUMPSELLER_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	fetcher := &loggedFetcher{test: t, client: source.Client{
		HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2,
		BaseDelay: time.Second, UserAgent: "muchi-api/1.0",
	}}
	client := jumpseller.Client{Domain: "www.deckscards.cl", Name: "Decks Cards", Fetcher: fetcher}
	started := time.Now()
	items, err := client.FindOffers(ctx, model.CardQuery{Name: "Dark Magician"})
	if err != nil || len(items) == 0 {
		t.Fatalf("requests=%d offers=%d duration=%s err=%v", fetcher.calls, len(items), time.Since(started), err)
	}
	for _, item := range items {
		if item.CardName != "Dark Magician" || item.VariantID == "" || item.StockStatus != "available" {
			t.Fatalf("unexpected offer=%+v", item)
		}
		t.Logf("variant=%s language=%s price=%s %s url=%s", item.VariantID, item.Language, item.PriceAmount, item.PriceCurrency, item.URL)
	}
	t.Logf("requests=%d offers=%d duration=%s", fetcher.calls, len(items), time.Since(started))
}

// loggedFetcher Reports Progress during Long Storefront Searches.
type loggedFetcher struct {
	test   *testing.T
	client source.Client
	calls  int
}

func (f *loggedFetcher) FetchSource(ctx context.Context, domain, target string) ([]byte, error) {
	f.calls++
	if f.calls%20 == 0 {
		f.test.Logf("%d Requests: %s", f.calls, target)
	}
	return f.client.FetchSource(ctx, domain, target)
}

func (f *loggedFetcher) FetchStorefront(ctx context.Context, domain, target string) ([]byte, error) {
	f.calls++
	if f.calls%20 == 0 {
		f.test.Logf("%d Requests: %s", f.calls, target)
	}
	return f.client.FetchStorefront(ctx, domain, target)
}
