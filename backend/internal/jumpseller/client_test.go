package jumpseller

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type fixtureFetcher func(string) ([]byte, error)

func (f fixtureFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	return f(target)
}

func TestStoreProducts(t *testing.T) {
	for _, test := range []struct{ domain, path, price, status string }{
		{"www.cartaslafortaleza.cl", "/sol-ring-ingles-nm-cma", "3000", "available"},
		{"www.chronomagic.cl", "/sol-ring-lcc-313-en-nm", "3000", "available"},
		{"gamequest.cl", "/sol-ring-espanol-nm-afc", "3200", "available"},
		{"www.magic4ever.cl", "/sol-ring-25", "2000", "unavailable"},
	} {
		t.Run(test.domain, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + test.domain + ".html")
			if err != nil {
				t.Fatal(err)
			}
			items, err := (Client{Domain: test.domain}).parseProduct(data, test.path)
			if err != nil || len(items) != 1 {
				t.Fatalf("items=%+v err=%v", items, err)
			}
			if items[0].PriceAmount != test.price || items[0].PriceCurrency != "CLP" || items[0].StockStatus != test.status {
				t.Fatalf("offer=%+v", items[0])
			}
			if test.domain == "www.magic4ever.cl" && (items[0].VariantID != "120640277" || items[0].Language != "Inglés" || items[0].Finish != "No Foil" || items[0].Condition != "NM") {
				t.Fatalf("variant=%+v", items[0])
			}
		})
	}
}

func TestPaginatedSearch(t *testing.T) {
	product, err := os.ReadFile("testdata/www.cartaslafortaleza.cl.html")
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := Client{Domain: "www.cartaslafortaleza.cl", Fetcher: fixtureFetcher(func(target string) ([]byte, error) {
		u, _ := url.Parse(target)
		if u.Path != "/api/search/Sol Ring" {
			calls++
			return product, nil
		}

		switch u.Query().Get("page") {
		case "1":
			return []byte(`{"products":[{"id":1,"name":"Other","permalink":"other"}]}`), nil
		case "2":
			return []byte(`{"products":[{"id":4,"name":"Other","permalink":"other"}]}`), nil
		case "3":
			return []byte(`{"products":[{"id":2,"name":"Sol Ring | Inglés | NM | CMA","permalink":"sol-ring-ingles-nm-cma"},{"id":3,"name":"Sol Ring Token","permalink":"token"}]}`), nil
		default:
			return []byte(`{"products":[]}`), nil
		}

	})}
	items, err := client.FindOffers(context.Background(), "Sol Ring")
	if err != nil || len(items) != 1 || calls != 1 {
		t.Fatalf("items=%v calls=%d err=%v", items, calls, err)
	}
}

func TestVariantStock(t *testing.T) {
	data, err := os.ReadFile("testdata/www.magic4ever.cl.html")
	if err != nil {
		t.Fatal(err)
	}
	client := Client{Domain: "www.magic4ever.cl", Fetcher: fixtureFetcher(func(string) ([]byte, error) { return data, nil })}
	status, err := client.CheckStock(context.Background(), offer.Offer{URL: "https://www.magic4ever.cl/sol-ring-25?variant_id=120640277"})
	if err != nil || status != "unavailable" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	status, err = client.CheckStock(context.Background(), offer.Offer{URL: "https://www.magic4ever.cl/sol-ring-25?variant_id=99"})
	if err != nil || status != "unknown" {
		t.Fatalf("status=%s err=%v", status, err)
	}
	if _, err = client.CheckStock(context.Background(), offer.Offer{URL: "https://evil.test/sol-ring"}); err == nil {
		t.Fatal("foreign URL accepted")
	}
}

func TestInvalidData(t *testing.T) {
	if _, err := (Client{}).parseProduct([]byte(`<html>Challenge</html>`), "/ring"); err == nil {
		t.Fatal("challenge accepted")
	}
	data, _ := os.ReadFile("testdata/www.cartaslafortaleza.cl.html")
	for _, broken := range []string{strings.ReplaceAll(string(data), "CLP", ""), strings.ReplaceAll(string(data), "3000.0", "null"), strings.ReplaceAll(string(data), `[]`, `null`)} {
		if _, err := (Client{Domain: "www.cartaslafortaleza.cl"}).parseProduct([]byte(broken), "/sol-ring-ingles-nm-cma"); err == nil {
			t.Fatal("invalid data accepted")
		}
	}
	failure := errors.New("HTTP 429")
	client := Client{Fetcher: fixtureFetcher(func(string) ([]byte, error) { return nil, failure })}
	if _, err := client.FindOffers(context.Background(), "Sol Ring"); !errors.Is(err, failure) {
		t.Fatalf("err=%v", err)
	}
	client.Fetcher = fixtureFetcher(func(string) ([]byte, error) {
		return []byte(`{"products":[{"id":1,"name":"Other","permalink":"other"}]}`), nil
	})
	if _, err := client.FindOffers(context.Background(), "Sol Ring"); err == nil {
		t.Fatal("repeated page accepted")
	}
	for _, pair := range [][3]string{{"1000", "100", "900"}, {"12.50", "0.25", "12.25"}} {
		got, err := priceAmount(json.Number(pair[0]), json.Number(pair[1]))
		if err != nil || got != pair[2] {
			t.Fatalf("amount=%s err=%v", got, err)
		}
	}
}

func TestMixedVariants(t *testing.T) {
	data, err := os.ReadFile("testdata/www.magic4ever.cl.html")
	if err != nil {
		t.Fatal(err)
	}
	form, currency, variants, err := readProduct(data, "/sol-ring-25", "www.magic4ever.cl")
	if err != nil {
		t.Fatal(err)
	}
	available := variants[0]
	available.Variant.ID++
	quantity := 1
	available.Variant.Stock = &quantity
	available.Price = "2400.50"
	available.Discount = "100.25"
	variants = append(variants, available)
	items, err := buildVariants(offer.Offer{URL: "https://www.magic4ever.cl/sol-ring-25", Source: "www.magic4ever.cl", PriceCurrency: currency}, form, variants)
	if err != nil || len(items) != 2 {
		t.Fatalf("items=%v err=%v", items, err)
	}
	if items[0].StockStatus != "unavailable" || items[1].StockStatus != "available" || items[1].PriceAmount != "2300.25" || items[0].URL == items[1].URL {
		t.Fatalf("items=%+v", items)
	}
	available.Status = "disabled"
	items, err = buildVariants(offer.Offer{}, form, []variantReply{available})
	if err != nil || items[0].StockStatus != "unavailable" {
		t.Fatalf("items=%v err=%v", items, err)
	}
}

func TestCancelledSearch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := Client{Fetcher: fixtureFetcher(func(string) ([]byte, error) { t.Fatal("request after cancellation"); return nil, nil })}
	if _, err := client.FindOffers(ctx, "Sol Ring"); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
