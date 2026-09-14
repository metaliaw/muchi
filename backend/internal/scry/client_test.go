package scry

import (
	"context"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
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

type printProvider struct{ prints []cardmetadata.Print }

func (p printProvider) CardPrints(context.Context, string) ([]cardmetadata.Print, error) {
	return p.prints, nil
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

func TestOffersCarryPrintImages(t *testing.T) {
	page := `<div id="results">
		<a data-track-type="store_offer_click" data-store-name="One" data-card-name="Sol Ring" data-variant-key="one" data-price-clp="2800" data-product-url="https://one.test/c21-263" data-offer-title="Sol Ring [C21] #263 ENG"></a>
		<a data-track-type="store_offer_click" data-store-name="Two" data-card-name="Sol Ring" data-variant-key="two" data-price-clp="2900" data-product-url="https://two.test/c21" data-offer-title="Sol Ring [C21] ENG"></a>
		<a data-track-type="store_offer_click" data-store-name="Three" data-card-name="Sol Ring" data-variant-key="three" data-price-clp="3000" data-product-url="https://three.test/soc" data-offer-title="Sol Ring [SOC] ENG"></a>
	</div>`
	client := Client{
		Fetcher: &pageFetcher{data: []byte(page)}, BaseURL: "https://scry.test",
		Prints: printProvider{prints: []cardmetadata.Print{
			{Edition: "c21", CollectorNumber: "263", Image: "https://images.test/c21-263.jpg"},
			{Edition: "c21", CollectorNumber: "264", Image: "https://images.test/c21-264.jpg"},
			{Edition: "soc", CollectorNumber: "1", Image: "https://images.test/soc-1.jpg"},
		}},
	}

	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})

	if err != nil || len(items) != 3 {
		t.Fatalf("FindOffers() items=%#v err=%v", items, err)
	}
	if items[0].Image != "https://images.test/c21-263.jpg" {
		t.Fatalf("exact image=%q", items[0].Image)
	}
	if items[1].Image != "https://images.test/c21-263.jpg" {
		t.Fatalf("first print image=%q", items[1].Image)
	}
	if items[2].Image != "https://images.test/soc-1.jpg" {
		t.Fatalf("edition image=%q", items[2].Image)
	}
}

func TestOffersMatchEditionsNamedInWords(t *testing.T) {
	page := `<div id="results">
		<a data-track-type="store_offer_click" data-store-name="One" data-card-name="Sol Ring" data-variant-key="one" data-price-clp="2800" data-product-url="https://one.test/pip" data-offer-title="Sol Ring [Fallout]"></a>
		<a data-track-type="store_offer_click" data-store-name="Two" data-card-name="Sol Ring" data-variant-key="two" data-price-clp="2900" data-product-url="https://two.test/mh2" data-offer-title="Sol Ring [Modern Horizons 2] #400"></a>
	</div>`
	client := Client{
		Fetcher: &pageFetcher{data: []byte(page)}, BaseURL: "https://scry.test",
		Prints: printProvider{prints: []cardmetadata.Print{
			{Edition: "pip", EditionName: "Fallout", CollectorNumber: "123", Image: "https://images.test/pip-123.jpg"},
			{Edition: "mh2", EditionName: "Modern Horizons 2", CollectorNumber: "400", Image: "https://images.test/mh2-400.jpg"},
		}},
	}

	items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})

	if err != nil || len(items) != 2 {
		t.Fatalf("FindOffers() items=%#v err=%v", items, err)
	}
	if items[0].Image != "https://images.test/pip-123.jpg" {
		t.Fatalf("edition by name image=%q", items[0].Image)
	}
	if items[1].Image != "https://images.test/mh2-400.jpg" {
		t.Fatalf("multiword edition image=%q", items[1].Image)
	}
}

// Cada Tienda Escribe la Impresión a su Manera. Estas son las Formas
// reales que scry.cl Publica, tomadas de una Búsqueda en Producción.
func TestOffersMatchEveryStoreTitleShape(t *testing.T) {
	prints := []cardmetadata.Print{
		{Edition: "c19", EditionName: "Commander 2019", CollectorNumber: "221", Image: "https://images.test/c19-221.jpg"},
		{Edition: "c14", EditionName: "Commander 2014", CollectorNumber: "270", Image: "https://images.test/c14-270.jpg"},
		{Edition: "mkc", EditionName: "Murders at Karlov Manor Commander", CollectorNumber: "237", Image: "https://images.test/mkc-237.jpg"},
		{Edition: "znc", EditionName: "Zendikar Rising Commander", CollectorNumber: "72", Image: "https://images.test/znc-72.jpg"},
		{Edition: "pip", EditionName: "Fallout", CollectorNumber: "123", Image: "https://images.test/pip-123.jpg"},
	}
	for _, test := range []struct{ name, title, want string }{
		{"code and hash number", "Sol Ring [C19] #221 ENG", "https://images.test/c19-221.jpg"},
		{"code and number in parentheses", "Sol Ring — Commander 2019 (C19 #221) · NM · EN", "https://images.test/c19-221.jpg"},
		{"code joined by a dash", "Sol Ring [C14-270] [Near Mint / Inglés]", "https://images.test/c14-270.jpg"},
		{"code spaced from its number", "Sol Ring [MKC - 237] [Moderately Played]", "https://images.test/mkc-237.jpg"},
		{"code among pipes", "Sol Ring | Inglés | NM | MKC", "https://images.test/mkc-237.jpg"},
		{"code buried in words", "Sol Ring [ZNC Normal Inglés NM Normal]", "https://images.test/znc-72.jpg"},
		{"edition named in words", "Sol Ring [Fallout]", "https://images.test/pip-123.jpg"},
		{"no edition at all", "Sol Ring — Near Mint", ""},
		{"condition only", "Sol Ring [Idioma: Español, Estado: NM]", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			page := `<div id="results"><a data-track-type="store_offer_click" data-store-name="One" data-card-name="Sol Ring" data-variant-key="one" data-price-clp="2800" data-product-url="https://one.test/x" data-offer-title="` + test.title + `"></a></div>`
			client := Client{Fetcher: &pageFetcher{data: []byte(page)}, BaseURL: "https://scry.test", Prints: printProvider{prints: prints}}

			items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})

			if err != nil || len(items) != 1 {
				t.Fatalf("FindOffers() items=%#v err=%v", items, err)
			}
			if items[0].Image != test.want {
				t.Fatalf("image=%q want %q", items[0].Image, test.want)
			}
		})
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
