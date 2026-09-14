package tcgmatch_test

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/tcgmatch"
)

func TestLiveDarkMagician(t *testing.T) {
	if os.Getenv("MUCHI_TEST_TCGMATCH_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_TCGMATCH_LIVE=1")
	}
	fetcher := &measuredFetcher{client: source.Client{
		HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2,
		BaseDelay: time.Second, UserAgent: "muchi-api/1.0",
	}}
	client := tcgmatch.Client{Fetcher: fetcher, BaseURL: "https://api.tcgmatch.cl", Game: "yugioh"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	started := time.Now()
	items, err := client.Search(ctx, "Dark Magician")
	if err != nil {
		t.Fatalf("requests=%d bytes=%d duration=%s err=%v", fetcher.calls, fetcher.bytes, time.Since(started), err)
	}
	seen := make(map[string]bool, len(items))
	duplicates := 0
	for _, item := range items {
		key := strings.Join([]string{item.Store, item.URL, item.VariantID}, "|")
		if seen[key] {
			duplicates++
		}
		seen[key] = true
		t.Logf("offer=%s store=%s price=%s %s language=%s", item.CardName, item.Store, item.PriceAmount, item.PriceCurrency, item.Language)
	}
	t.Logf("requests=%d bytes=%d offers=%d duplicates=%d duration=%s", fetcher.calls, fetcher.bytes, len(items), duplicates, time.Since(started))
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
