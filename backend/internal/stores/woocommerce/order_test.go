package woocommerce

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// TestPlaceOrderRefusesThePlaceholderBuyer Keeps a real Order from ever
// Landing on the reserved, undeliverable Placeholder Mailbox: PlaceOrder
// Refuses before any Call, until BuyerEmail Names a real one.
func TestPlaceOrderRefusesThePlaceholderBuyer(t *testing.T) {
	sessions := &cartSessions{}
	client := Client{Domain: "woo.test", Sessions: sessions}
	ring := model.Offer{ID: "ring", Source: "woo.test", VariantID: "7"}
	_, err := client.PlaceOrder(context.Background(), []model.CartLine{{Offer: ring, Quantity: 1}}, model.ShippingAddress{Country: "CL"})
	if err != ErrVirtualBuyerNotConfigured {
		t.Fatalf("err = %v, want ErrVirtualBuyerNotConfigured", err)
	}
	if len(sessions.calls) != 0 {
		t.Fatalf("calls = %+v, want none before the buyer is real", sessions.calls)
	}
}

// checkoutSessions Plays a Store API that Selects the flat Rate on its own
// and Confirms an Order pending Payment.
type checkoutSessions struct{ calls []source.SessionRequest }

func (c *checkoutSessions) SendSession(_ context.Context, _ string, request source.SessionRequest) ([]byte, http.Header, error) {
	c.calls = append(c.calls, request)
	switch {
	case request.Method == http.MethodGet:
		return []byte(`{}`), http.Header{"Cart-Token": {"tok"}}, nil
	case strings.HasSuffix(request.Target, "/select-shipping-rate"):
		return []byte(`{}`), nil, nil
	case strings.HasSuffix(request.Target, "/checkout"):
		return []byte(`{"order_id":501,"status":"pending","payment_result":{"payment_status":"pending","redirect_url":""}}`), nil, nil
	default:
		return []byte(`{
			"items":[{"id":7,"quantity":1,"prices":{"price":"3000","currency_minor_unit":0}}],
			"totals":{"total_items":"3000","total_shipping":"3990","total_price":"6990","currency_code":"CLP","currency_minor_unit":0},
			"shipping_rates":[{"shipping_rates":[{"rate_id":"flat_rate:1","name":"Starken","price":"3990","selected":true}]}],
			"payment_methods":["bacs"]}`), nil, nil
	}
}

// TestPlaceOrderChecksOutByBankTransferOnceTheBuyerIsReal Spells the Path a
// configured Buyer Unlocks: pick the Cart's own Rate, Checkout by `bacs`,
// and Answer the Order the Store Confirmed.
func TestPlaceOrderChecksOutByBankTransferOnceTheBuyerIsReal(t *testing.T) {
	sessions := &checkoutSessions{}
	client := Client{Domain: "woo.test", Name: "Woo", Sessions: sessions, BuyerName: "Muchi", BuyerEmail: "pedidos@muchi.cl"}
	ring := model.Offer{ID: "ring", Source: "woo.test", VariantID: "7"}
	order, err := client.PlaceOrder(context.Background(), []model.CartLine{{Offer: ring, Quantity: 1}}, model.ShippingAddress{Country: "CL"})
	if err != nil {
		t.Fatal(err)
	}
	if order.Store != "Woo" || order.Domain != "woo.test" || order.Status != model.OrderPending || order.StoreOrder != "501" {
		t.Fatalf("order = %+v", order)
	}
	var sawRate, sawCheckout bool
	for _, call := range sessions.calls {
		if strings.HasSuffix(call.Target, "/select-shipping-rate") {
			sawRate = true
			if !strings.Contains(string(call.Body), `"rate_id":"flat_rate:1"`) {
				t.Fatalf("rate body = %s", call.Body)
			}
		}
		if strings.HasSuffix(call.Target, "/checkout") {
			sawCheckout = true
			if !strings.Contains(string(call.Body), `"email":"pedidos@muchi.cl"`) || !strings.Contains(string(call.Body), `"payment_method":"bacs"`) {
				t.Fatalf("checkout body = %s", call.Body)
			}
		}
	}
	if !sawRate || !sawCheckout {
		t.Fatalf("calls = %+v", sessions.calls)
	}
}
