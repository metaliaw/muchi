package shopify

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// cartSessions Plays an Ajax Cart that Trims the second Variant to one Unit.
type cartSessions struct{ calls []source.SessionRequest }

func (c *cartSessions) SendSession(_ context.Context, _ string, request source.SessionRequest) ([]byte, http.Header, error) {
	c.calls = append(c.calls, request)
	switch {
	case strings.HasSuffix(request.Target, "/cart/add.js"):
		return []byte(`{}`), http.Header{"Set-Cookie": {"cart=abc; path=/; secure", "cart_sig=def; path=/"}}, nil
	case strings.HasSuffix(request.Target, "/cart.js"):
		return []byte(`{"currency":"CLP","total_price":650000,"items":[{"variant_id":11,"quantity":2,"final_price":300000},{"variant_id":22,"quantity":1,"final_price":50000}]}`), nil, nil
	default:
		return []byte(`{"shipping_rates":[{"code":"Starken","name":"Starken","price":"4990.00"},{"code":"Retiro","name":"Retiro en tienda","price":"0.00"}]}`), nil, nil
	}
}

func TestQuoteCartReadsCartAndCheapestRate(t *testing.T) {
	sessions := &cartSessions{}
	client := Client{Domain: "shop.test", Sessions: sessions}
	ring := offer.Offer{ID: "ring", URL: "https://shop.test/products/sol-ring?variant=11"}
	bolt := offer.Offer{ID: "bolt", Source: "shop.test", VariantID: "22"}
	quote, err := client.QuoteCart(context.Background(), []offer.CartLine{{Offer: ring, Quantity: 1}, {Offer: ring, Quantity: 1}, {Offer: bolt, Quantity: 3}}, offer.ShippingAddress{Country: "CL", Region: "RM"})
	if err != nil {
		t.Fatal(err)
	}
	if quote.Items != "6500" || quote.Shipping != "0" || quote.Total != "6500" || quote.Currency != "CLP" || len(quote.ShippingRates) != 2 || quote.ShippingRates[0].Price != "4990" {
		t.Fatalf("quote = %+v", quote)
	}
	if len(quote.Notices) != 1 || quote.Notices[0] != "bolt: asked 3, cart holds 1" {
		t.Fatalf("notices = %v", quote.Notices)
	}
	if cookie := sessions.calls[1].Header.Get("Cookie"); cookie != "cart=abc; cart_sig=def" {
		t.Fatalf("cookie = %q", cookie)
	}
	if rates := sessions.calls[2].Target; !strings.Contains(rates, "shipping_address%5Bprovince%5D=RM") || !strings.Contains(rates, "shipping_address%5Bcountry%5D=CL") {
		t.Fatalf("rates target = %s", rates)
	}
}

func TestQuoteCartRefusesAggregatorKeys(t *testing.T) {
	sessions := &cartSessions{}
	item := offer.Offer{URL: "https://shop.test/products/bolt", VariantID: "99", Source: "scry.cl"}
	if _, err := (Client{Domain: "shop.test", Sessions: sessions}).QuoteCart(context.Background(), []offer.CartLine{{Offer: item, Quantity: 1}}, offer.ShippingAddress{Country: "CL"}); err != offer.ErrNoQuote || len(sessions.calls) != 0 {
		t.Fatalf("err = %v calls = %d", err, len(sessions.calls))
	}
}
