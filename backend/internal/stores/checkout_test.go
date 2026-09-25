package stores

import (
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TestCheckoutLinkNeedsAShopifyVariantForEveryLine Keeps a Buyer away from a
// Cart Missing a Card: one Line without its Variant Drops the whole Link.
func TestCheckoutLinkNeedsAShopifyVariantForEveryLine(t *testing.T) {
	config := Config{Stores: map[string]StoreConfig{
		"shop.test":  {Platform: "shopify", Enabled: true},
		"off.test":   {Platform: "shopify"},
		"jumps.test": {Platform: "jumpseller", Enabled: true},
	}}
	ring := model.Offer{URL: "https://shop.test/products/sol-ring?variant=11"}
	// scry.cl Sends its own Key; it is not a Shopify Variant.
	aggregated := model.Offer{URL: "https://shop.test/products/bolt", VariantID: "99", Source: "scry.cl"}
	cases := []struct {
		name   string
		domain string
		lines  []model.CartLine
		want   string
	}{
		{"variant in url", "shop.test", []model.CartLine{{Offer: ring, Quantity: 3}}, "https://shop.test/cart/11:3"},
		{"same variant twice adds up", "shop.test", []model.CartLine{{Offer: ring, Quantity: 1}, {Offer: ring, Quantity: 2}}, "https://shop.test/cart/11:3"},
		{"aggregator key is not trusted", "shop.test", []model.CartLine{{Offer: ring, Quantity: 1}, {Offer: aggregated, Quantity: 1}}, ""},
		{"disabled store", "off.test", []model.CartLine{{Offer: model.Offer{URL: "https://off.test/p?variant=1"}, Quantity: 1}}, ""},
		{"platform without permalink", "jumps.test", []model.CartLine{{Offer: model.Offer{URL: "https://jumps.test/p?variant=1"}, Quantity: 1}}, ""},
		{"unknown store", "nowhere.test", []model.CartLine{{Offer: ring, Quantity: 1}}, ""},
	}
	for _, test := range cases {
		link, ok := config.CheckoutLink(test.domain, test.lines)
		if link != test.want || ok != (test.want != "") {
			t.Errorf("%s: link = %q, %v; want %q", test.name, link, ok, test.want)
		}
	}
}
