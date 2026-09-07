package stores

import (
	"context"
	"strings"
	"testing"
)

type catalogFetcher struct{ target string }

func (f *catalogFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.target = target
	return []byte(`[{"id":35521,"name":"Sol Ring","permalink":"https://lacripta.cl/producto/sol-ring-8/","is_in_stock":true,"prices":{"price":"3000","currency_code":"CLP","currency_minor_unit":0}},{"name":"Sol Ring Token"}]`), nil
}
func TestCatalogOffers(t *testing.T) {
	fetcher := &catalogFetcher{}
	items, err := (Catalog{Fetcher: fetcher, Domain: "lacripta.cl"}).FindOffers(context.Background(), "Sol Ring")
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].PriceAmount != "3000" || items[0].Source != "lacripta.cl" || items[0].StockStatus != "available" || !strings.Contains(fetcher.target, "search=Sol+Ring") {
		t.Fatalf("offer=%+v request=%s", items[0], fetcher.target)
	}
}
func TestFormatPrice(t *testing.T) {
	for _, test := range []struct {
		value string
		minor int
		want  string
	}{{"3000", 0, "3000"}, {"1299", 2, "12.99"}, {"1", 2, "0.01"}} {
		got, err := formatPrice(test.value, test.minor)
		if err != nil || got != test.want {
			t.Fatalf("price=%s err=%v", got, err)
		}
	}
	if _, err := formatPrice("unknown", 0); err == nil {
		t.Fatal("expected invalid price")
	}
}
