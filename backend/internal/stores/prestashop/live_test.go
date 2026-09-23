package prestashop_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores/prestashop"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

func TestLiveDarkMagician(t *testing.T) {
	if os.Getenv("MUCHI_TEST_PRESTASHOP_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_PRESTASHOP_LIVE=1")
	}
	fetcher := &measuredFetcher{client: source.Client{
		HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2,
		BaseDelay: time.Second, UserAgent: "muchi-api/1.0",
	}}
	client := prestashop.Client{
		Fetcher: fetcher, Domain: "v3.netdecker.cl", Name: "Netdecker",
		SearchPath: "/busqueda", Currency: "CLP",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	started := time.Now()
	items, err := client.FindOffers(ctx, offer.CardQuery{Name: "Dark Magician"})
	duration := time.Since(started)
	if err != nil {
		t.Fatalf("pages=%d bytes=%d duration=%s err=%v", fetcher.calls, fetcher.bytes, duration, err)
	}
	t.Logf("pages=%d bytes=%d offers=%d duration=%s", fetcher.calls, fetcher.bytes, len(items), duration)
	for _, item := range items {
		t.Logf("offer=%s price=%s %s stock=%s url=%s", item.CardName, item.PriceAmount, item.PriceCurrency, item.StockStatus, item.URL)
	}
}

type measuredFetcher struct {
	client source.Client
	calls  int
	bytes  int
}

func (f *measuredFetcher) FetchSource(ctx context.Context, domain, target string) ([]byte, error) {
	data, err := f.client.FetchSource(ctx, domain, target)
	f.calls++
	f.bytes += len(data)
	return data, err
}
