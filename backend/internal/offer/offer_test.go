package offer

import "testing"

func TestNormalizeOffers(t *testing.T) {
	items := []Offer{
		{Store: "one", URL: "https://one.test/card", VariantID: "a"},
		{Store: "one", URL: "https://one.test/card", VariantID: "a"},
	}
	if got := NormalizeCard("  Sol   RING "); got != "sol ring" {
		t.Fatalf("NormalizeCard() = %q", got)
	}
	if got := DeduplicateOffers(items); len(got) != 1 {
		t.Fatalf("DeduplicateOffers() returned %d items", len(got))
	}
}

func TestSelectOffers(t *testing.T) {
	items := []Offer{
		{ID: "cheap", PriceAmount: "1.00", PriceCurrency: "USD"},
		{ID: "middle", PriceAmount: "10.00", PriceCurrency: "USD"},
		{ID: "high", PriceAmount: "11.00", PriceCurrency: "USD"},
	}
	items = MarkSuspicious(items)
	if !items[0].Suspicious || items[0].SuspiciousReason == "" {
		t.Fatal("cheap offer was not marked suspicious")
	}
	selected := SelectStockOffers(items, 5)
	if len(selected) != 2 || selected[0].ID != "middle" {
		t.Fatalf("SelectStockOffers() = %#v", selected)
	}
}

func TestRejectOffer(t *testing.T) {
	valid := Offer{Store: "one", Source: "catalog", PriceAmount: "10.00", URL: "https://one.test/card"}
	if err := ValidateOffer(valid); err != nil {
		t.Fatalf("ValidateOffer() error = %v", err)
	}
	valid.URL = "/relative"
	if err := ValidateOffer(valid); err == nil {
		t.Fatal("ValidateOffer() accepted relative URL")
	}
}
