package woocommerce

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// cartSessions Plays a Store API Cart that Trims the second Product to one Unit.
type cartSessions struct{ calls []source.SessionRequest }

func (c *cartSessions) SendSession(_ context.Context, _ string, request source.SessionRequest) ([]byte, http.Header, error) {
	c.calls = append(c.calls, request)
	if request.Method == http.MethodGet {
		return []byte(`{}`), http.Header{"Cart-Token": {"tok"}}, nil
	}
	return []byte(`{
		"items":[{"id":7,"quantity":2,"prices":{"price":"3000","currency_minor_unit":0}},{"id":8,"quantity":1,"prices":{"price":"500","currency_minor_unit":0}}],
		"totals":{"total_items":"6500","total_shipping":"3990","total_price":"10490","currency_code":"CLP","currency_minor_unit":0},
		"shipping_rates":[{"shipping_rates":[{"rate_id":"flat_rate:1","name":"Starken &amp; más","price":"3990","selected":true},{"rate_id":"local_pickup:2","name":"Retiro","price":"0","selected":false}]}],
		"payment_methods":["bacs","transbank_webpay_plus_rest"]}`), nil, nil
}

func TestQuoteCartReadsTotalsShippingAndTrimmedLines(t *testing.T) {
	sessions := &cartSessions{}
	client := Client{Domain: "woo.test", Sessions: sessions}
	ring := offer.Offer{ID: "ring", Source: "woo.test", VariantID: "7"}
	bolt := offer.Offer{ID: "bolt", Source: "woo.test", VariantID: "8"}
	quote, err := client.QuoteCart(context.Background(), []offer.CartLine{{Offer: ring, Quantity: 2}, {Offer: bolt, Quantity: 3}}, offer.ShippingAddress{Country: "CL", Region: "CL-RM"})
	if err != nil {
		t.Fatal(err)
	}
	if quote.Total != "10490" || quote.Shipping != "3990" || quote.Items != "6500" || quote.Currency != "CLP" {
		t.Fatalf("totals = %+v", quote)
	}
	if len(quote.ShippingRates) != 2 || quote.ShippingRates[0].Name != "Starken & más" || quote.ShippingRates[1].Price != "0" {
		t.Fatalf("rates = %+v", quote.ShippingRates)
	}
	if len(quote.Notices) != 1 || !strings.Contains(quote.Notices[0], "bolt: asked 3, cart holds 1") {
		t.Fatalf("notices = %v", quote.Notices)
	}
	if quote.Lines[0].UnitPrice != "3000" || len(quote.PaymentMethods) != 2 {
		t.Fatalf("lines = %+v methods = %v", quote.Lines, quote.PaymentMethods)
	}
	// One read for the token, one add per line, one address; every write Carries the Cart.
	if len(sessions.calls) != 4 || !strings.HasSuffix(sessions.calls[3].Target, "/cart/update-customer") {
		t.Fatalf("calls = %+v", sessions.calls)
	}
	for _, call := range sessions.calls[1:] {
		if call.Header.Get("Cart-Token") != "tok" {
			t.Fatalf("call without cart token: %+v", call)
		}
	}
}

// TestQuoteCartRefusesOffersItCannotName Keeps an Aggregator's Key or a
// Variable Product out of a Store Cart before any Call is Made.
func TestQuoteCartRefusesOffersItCannotName(t *testing.T) {
	sessions := &cartSessions{}
	client := Client{Domain: "woo.test", Sessions: sessions}
	for _, item := range []offer.Offer{{Source: "scry.cl", VariantID: "7"}, {Source: "woo.test"}} {
		if _, err := client.QuoteCart(context.Background(), []offer.CartLine{{Offer: item, Quantity: 1}}, offer.ShippingAddress{Country: "CL"}); err != offer.ErrNoQuote {
			t.Fatalf("offer %+v err = %v", item, err)
		}
	}
	if len(sessions.calls) != 0 {
		t.Fatalf("calls = %d", len(sessions.calls))
	}
}
