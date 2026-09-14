package stores_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// TestKonohaLiveProbe Reads the real Store the Way Production Reads it.
func TestKonohaLiveProbe(t *testing.T) {
	if os.Getenv("MUCHI_TEST_STORES_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_STORES_LIVE=1")
	}
	client := stores.Catalog{
		Fetcher: source.Client{
			HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2,
			BaseDelay: time.Second, UserAgent: "muchi-api/1.0",
		},
		Domain: "konohastore.cl", Name: "Konoha Store",
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, card := range []string{"Kuriboh", "Dark Magician", "Ash Blossom & Joyous Spring", "Pot of Desires"} {
		items, err := client.FindOffers(ctx, offer.CardQuery{Name: card})
		if err != nil {
			t.Errorf("card=%q err=%v", card, err)
			continue
		}
		t.Logf("card=%q offers=%d", card, len(items))
		for _, item := range items {
			t.Logf("  %s | %s %s | %s | %s", item.CardName, item.PriceAmount, item.PriceCurrency, item.StockStatus, item.URL)
		}
	}
}
