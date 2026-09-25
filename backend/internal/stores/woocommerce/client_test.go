package woocommerce

import (
	"context"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type catalogFetcher struct{ target string }

func (f *catalogFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.target = target
	return []byte(`[{"id":35521,"name":"Sol Ring","type":"simple","permalink":"https://lacripta.cl/producto/sol-ring-8/","is_in_stock":true,"prices":{"price":"3000","currency_code":"CLP","currency_minor_unit":0}},{"name":"Sol Ring Token"}]`), nil
}
func TestCatalogOffers(t *testing.T) {
	fetcher := &catalogFetcher{}
	items, err := (Client{Fetcher: fetcher, Domain: "lacripta.cl"}).FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].PriceAmount != "3000" || items[0].Source != "lacripta.cl" || items[0].StockStatus != "available" || items[0].VariantID != "35521" || !strings.Contains(fetcher.target, "search=Sol+Ring") {
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

type quotedFetcher struct{}

func (quotedFetcher) FetchSource(_ context.Context, _, _ string) ([]byte, error) {
	return []byte(`[{"id":63392,"name":"MAMO-EN002 &#8220;Kuriboh&#8221; Ultra Rare Effect Monster","permalink":"https://konohastore.cl/producto/mamo-en002/","is_in_stock":true,"prices":{"price":"15000","currency_code":"CLP","currency_minor_unit":0}},{"id":31035,"name":"LDS3-EN100 &#8220;Winged Kuriboh&#8221; Common","permalink":"https://konohastore.cl/producto/lds3-en100/","is_in_stock":true,"prices":{"price":"400","currency_code":"CLP","currency_minor_unit":0}}]`), nil
}

// TestCatalogReadsQuotedTitle Covers a Store that Leads with its Set Code.
func TestCatalogReadsQuotedTitle(t *testing.T) {
	items, err := (Client{Fetcher: quotedFetcher{}, Domain: "konohastore.cl", Name: "Konoha Store"}).FindOffers(context.Background(), offer.CardQuery{Name: "Kuriboh"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].CardName != "MAMO-EN002 “Kuriboh” Ultra Rare Effect Monster" || items[0].PriceAmount != "15000" {
		t.Fatalf("offer=%+v", items[0])
	}
}
