package scry

import (
	"context"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/source"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type pageFetcher struct {
	domain, target string
	data           []byte
	err            error
}

func (f *pageFetcher) FetchSource(_ context.Context, domain, target string) ([]byte, error) {
	f.domain, f.target = domain, target
	return f.data, f.err
}

const offerPage = `<div id="results"><a data-track-type="store_offer_click" data-store-name="CatLotus" data-card-name="Sol Ring" data-variant-key="catlotus:45778" data-price-clp="2800" data-product-url="https://catlotus.cl/cartas/C21/Sol%20Ring?name=Sol+Ring&amp;set=C21" data-offer-title="Sol Ring [C21] ENG">Ver en tienda</a></div>`

func TestFindOffers(t *testing.T) {
	fetcher := &pageFetcher{data: []byte(offerPage)}
	client := Client{Fetcher: fetcher, BaseURL: "https://scry.cl/"}
	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})
	if err != nil || len(items) != 1 {
		t.Fatalf("offers: %v, %v", items, err)
	}
	item := items[0]
	if fetcher.domain != "scry.cl" || fetcher.target != "https://scry.cl/card/sol-ring" {
		t.Fatalf("request: %+v", fetcher)
	}
	if item.Source != "scry.cl" || item.PriceCurrency != "CLP" || item.PriceAmount != "2800" || item.VariantID != "catlotus:45778" || item.StockStatus != "unknown" || !strings.Contains(item.URL, "&set=C21") {
		t.Fatalf("offer: %+v", item)
	}
}

func TestPageFailures(t *testing.T) {
	for _, test := range []struct {
		name, page string
		wantError  bool
	}{
		{"empty", `<div id="results"></div>`, false},
		{"unexpected", `<html>Unavailable</html>`, true},
		{"invalid price", strings.Replace(offerPage, `data-price-clp="2800"`, `data-price-clp="unknown"`, 1), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := readOffers([]byte(test.page))
			if (err != nil) != test.wantError {
				t.Fatalf("error: %v", err)
			}
		})
	}
	fetcher := &pageFetcher{err: source.StatusError{Code: 404}}
	items, err := (Client{Fetcher: fetcher, BaseURL: "https://scry.cl"}).FindOffers(context.Background(), offer.CardQuery{Name: "Missing"})
	if err != nil || len(items) != 0 {
		t.Fatalf("missing card: %v, %v", items, err)
	}
	fetcher.err = source.StatusError{Code: 503}
	if _, err := (Client{Fetcher: fetcher, BaseURL: "https://scry.cl"}).FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"}); err == nil {
		t.Fatal("expected upstream error")
	}
}

func TestCardSlug(t *testing.T) {
	if got := buildCardSlug("  Éowyn, Shieldmaiden // Test  "); got != "eowyn-shieldmaiden-test" {
		t.Fatal(got)
	}
}
