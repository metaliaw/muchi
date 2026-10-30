package commerce

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

type variantFetcher struct{ target string }

func (f *variantFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.target = target
	return []byte(`{"id":1,"title":"Sol Ring","variants":[{"id":10,"available":true},{"id":11,"available":false}]}`), nil
}
func TestCheckShopifyVariant(t *testing.T) {
	fetcher := &variantFetcher{}
	checker := Shops{Fetcher: fetcher, Config: directory.Config{Stores: map[string]directory.StoreConfig{"cards.test": {Platform: "shopify", Enabled: true}}}}
	reading, err := checker.CheckStock(context.Background(), model.Offer{URL: "https://cards.test/products/ring?variant=11", VariantID: "10"})
	if err != nil || reading.Status != "unavailable" || fetcher.target != "https://cards.test/products/ring.js" {
		t.Fatalf("status=%s target=%s err=%v", reading.Status, fetcher.target, err)
	}
}
