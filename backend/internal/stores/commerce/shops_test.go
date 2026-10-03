package commerce

import (
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

// TestEachPlatformOffersOnlyWhatItCanDo Pins the one Table a new Platform
// Joins: which Capabilities its Client Answers for.
func TestEachPlatformOffersOnlyWhatItCanDo(t *testing.T) {
	shops := Shops{Config: directory.Config{Stores: map[string]directory.StoreConfig{
		"js.test":    {Platform: "jumpseller", Enabled: true},
		"shop.test":  {Platform: "shopify", Enabled: true},
		"woo.test":   {Platform: "woocommerce", Enabled: true},
		"ps.test":    {Platform: "prestashop", Enabled: true},
		"other.test": {Platform: "magento", Enabled: true},
	}}}
	cases := []struct {
		domain                              string
		finds, stocks, quotes, orders, link bool
	}{
		{"js.test", true, true, false, false, false},
		{"shop.test", true, true, true, false, true},
		{"woo.test", true, false, true, true, true},
		{"ps.test", true, false, false, false, false},
		{"other.test", false, false, false, false, false},
	}
	for _, test := range cases {
		shop, found := shops.Shop(test.domain)
		if !found {
			t.Fatalf("%s: an enabled store was not found", test.domain)
		}
		_, finds := shop.Client.(OfferFinder)
		_, stocks := shop.Client.(StockReader)
		_, quotes := shop.Client.(cartQuoter)
		_, orders := shop.Client.(orderPlacer)
		_, link := shop.Client.(cartLinker)
		got := [5]bool{finds, stocks, quotes, orders, link}
		want := [5]bool{test.finds, test.stocks, test.quotes, test.orders, test.link}
		if got != want {
			t.Errorf("%s: finds/stocks/quotes/orders/link = %v, want %v", test.domain, got, want)
		}
	}
}

// TestADisabledOrUnknownStoreIsNeverOpened Keeps every Question away from a
// Store that has not Opted in.
func TestADisabledOrUnknownStoreIsNeverOpened(t *testing.T) {
	shops := Shops{Config: directory.Config{Stores: map[string]directory.StoreConfig{
		"off.test": {Platform: "shopify"},
	}}}
	for _, domain := range []string{"off.test", "nowhere.test"} {
		if _, found := shops.Shop(domain); found {
			t.Errorf("%s: opened", domain)
		}
		if _, found := shops.Finder(domain); found {
			t.Errorf("%s: offered a finder", domain)
		}
	}
}
