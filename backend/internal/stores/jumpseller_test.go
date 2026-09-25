package stores

import (
	"context"
	"os"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type productFetcher struct{ target string }

func (f *productFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.target = target
	return os.ReadFile("jumpseller/testdata/www.magic4ever.cl.html")
}
func TestCheckJumpsellerVariant(t *testing.T) {
	fetcher := &productFetcher{}
	checker := Checker{Fetcher: fetcher, Config: Config{Stores: map[string]StoreConfig{"www.magic4ever.cl": {Platform: "jumpseller", Enabled: true}}}}
	reading, err := checker.CheckStock(context.Background(), model.Offer{URL: "https://www.magic4ever.cl/sol-ring-25?variant_id=120640277", VariantID: "99"})
	if err != nil || reading.Status != "unavailable" || fetcher.target != "https://www.magic4ever.cl/sol-ring-25" {
		t.Fatalf("status=%s target=%s err=%v", reading.Status, fetcher.target, err)
	}
}
