package stores

import (
	"context"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TestOrdererOnlyReachesAnEnabledWooCommerceStore Keeps PlaceOrder from ever
// Asking a Platform, or a disabled Store, that cannot Place one.
func TestOrdererOnlyReachesAnEnabledWooCommerceStore(t *testing.T) {
	sessions := &recordingSessions{}
	orderer := Orderer{Sessions: sessions, Config: Config{Stores: map[string]StoreConfig{
		"woo.test":     {Platform: "woocommerce", Enabled: true},
		"woo-off.test": {Platform: "woocommerce"},
		"shop.test":    {Platform: "shopify", Enabled: true},
	}}}
	lines := []model.CartLine{{Offer: model.Offer{Source: "woo.test", VariantID: "7"}, Quantity: 1}}
	address := model.ShippingAddress{Country: "CL"}
	cases := []struct{ domain string }{{"woo-off.test"}, {"shop.test"}, {"nowhere.test"}}
	for _, test := range cases {
		if _, err := orderer.PlaceOrder(context.Background(), test.domain, lines, address); err != model.ErrOrderNotSupported {
			t.Errorf("%s: err = %v, want ErrOrderNotSupported", test.domain, err)
		}
	}
	if len(sessions.calls) != 0 {
		t.Fatalf("calls = %+v, want none: every case above should refuse first", sessions.calls)
	}
	// The one enabled WooCommerce store still refuses, on the virtual-buyer
	// placeholder guard — proof the call actually reached PlaceOrder.
	if _, err := orderer.PlaceOrder(context.Background(), "woo.test", lines, address); err == nil || err == model.ErrOrderNotSupported {
		t.Fatalf("enabled woocommerce store err = %v", err)
	}
}

// TestOrdererResolvesTheAddressLikeQuoterDoes Keeps PlaceOrder from Sending
// WooCommerce the raw Buyer Spelling ("Chile", "Región Metropolitana")
// Quoter already Normalizes for the same Platform — a Mismatch here would
// have a live Store Validate or Rate the wrong Address.
func TestOrdererResolvesTheAddressLikeQuoterDoes(t *testing.T) {
	sessions := &recordingSessions{}
	orderer := Orderer{
		Sessions: sessions, BuyerEmail: "pedidos@muchi.cl",
		Config: Config{Stores: map[string]StoreConfig{"woo.test": {Platform: "woocommerce", Enabled: true}}},
	}
	lines := []model.CartLine{{Offer: model.Offer{Source: "woo.test", VariantID: "7"}, Quantity: 1}}
	address := model.ShippingAddress{Country: "Chile", Region: "Región Metropolitana"}
	// recordingSessions Answers an empty Cart, so PlaceOrder Fails past
	// fillCart for lack of a Shipping Rate — enough to Inspect the Address it
	// already Sent.
	if _, err := orderer.PlaceOrder(context.Background(), "woo.test", lines, address); err == nil {
		t.Fatal("want an error past fillCart, got nil")
	}
	if last := string(sessions.calls[len(sessions.calls)-1].Body); !strings.Contains(last, `"state":"CL-RM"`) || !strings.Contains(last, `"country":"CL"`) {
		t.Fatalf("address body = %s", last)
	}
}
