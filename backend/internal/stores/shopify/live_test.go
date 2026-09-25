package shopify_test

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

func TestLiveSolRing(t *testing.T) {
	if os.Getenv("MUCHI_TEST_SHOPIFY_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_SHOPIFY_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	for domain, store := range config.Stores {
		if store.Enabled && store.Platform == "shopify" {
			t.Run(domain, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
				defer cancel()
				client := shopify.Client{Domain: domain, Name: store.Name, Fetcher: source.Client{HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2, BaseDelay: time.Second, UserAgent: "muchi-api/1.0"}}
				items, err := client.FindOffers(ctx, model.CardQuery{Name: "Sol Ring"})
				if err != nil || len(items) == 0 {
					t.Fatalf("offers=%d err=%v", len(items), err)
				}
				for _, item := range items {
					if item.PriceCurrency != "CLP" || item.StockStatus != "available" {
						t.Fatalf("offer=%+v", item)
					}
				}
				reading, err := client.CheckStock(ctx, items[0])
				if err != nil || reading.Status != "available" {
					t.Fatalf("stock=%s err=%v", reading.Status, err)
				}
				t.Logf("Sol Ring: %d Offers; First Price: %s CLP; Variant Stock: %s", len(items), items[0].PriceAmount, reading.Status)
			})
		}
	}
}

func TestLiveOasisGames(t *testing.T) {
	if os.Getenv("MUCHI_TEST_SHOPIFY_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_SHOPIFY_LIVE=1")
	}
	client := shopify.Client{
		Domain: "www.oasisgames.cl", Name: "Oasis Games",
		Fetcher: source.Client{
			HTTP: &http.Client{Timeout: 15 * time.Second}, MaxAttempts: 2,
			BaseDelay: time.Second, UserAgent: "muchi-api/1.0",
		},
	}
	for _, name := range []string{"Sol Ring", "Pikachu ex"} {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		items, err := client.FindOffers(ctx, model.CardQuery{Name: name})
		cancel()
		if err != nil || len(items) == 0 {
			t.Fatalf("card=%q offers=%d err=%v", name, len(items), err)
		}
		for _, item := range items {
			if item.PriceCurrency != "CLP" || item.StockStatus != "available" || item.Image == "" {
				t.Fatalf("card=%q offer=%+v", name, item)
			}
		}
	}
}
