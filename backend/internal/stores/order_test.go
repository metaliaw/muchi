package stores

import (
	"context"
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
