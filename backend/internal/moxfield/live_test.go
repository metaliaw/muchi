package moxfield_test

import (
	"context"
	"encoding/json"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

type measuredCache struct{ count, size int }

func (c *measuredCache) LoadOffers(context.Context, string) ([]offer.Offer, bool, error) {
	return nil, false, nil
}
func (c *measuredCache) SaveOffers(_ context.Context, _ string, items []offer.Offer, _ time.Duration) error {
	data, err := json.Marshal(items)
	c.count, c.size = len(items), len(data)
	return err
}

func (c *measuredCache) DropOffers(context.Context, string) error { return nil }

// TestPublicLists is Opt-in: it Reads the Configured Live Inventories.
func TestPublicLists(t *testing.T) {
	if os.Getenv("MUCHI_TEST_MOXFIELD_LIVE") != "1" {
		t.Skip("set MUCHI_TEST_MOXFIELD_LIVE=1 to query public lists")
	}
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := source.Client{HTTP: &http.Client{Timeout: 10 * time.Second}, UserAgent: "muchi-api/1.0", MaxAttempts: 1, MaxBodyBytes: 16 << 20}
	for storeID, store := range config.Stores {
		if !store.Enabled {
			continue
		}
		for _, list := range store.Lists {
			t.Run(list.Label, func(t *testing.T) {
				cache := &measuredCache{}
				client := moxfield.Client{Cache: cache, TTL: time.Minute, Fetcher: fetcher, StoreID: storeID, Store: store.Name, Label: list.Label, ListURL: list.URL, Rate: list.CLPPerCKUSD}
				items, err := client.FindOffers(context.Background(), offer.CardQuery{Name: "Sol Ring"})
				if err != nil {
					t.Fatal(err)
				}
				if cache.size >= 1_000_000 {
					t.Fatalf("inventory exceeds Firestore payload budget: %d bytes", cache.size)
				}
				t.Logf("Resolved %s: %d variants, %d cache bytes; Sol Ring offers: %d", list.Label, cache.count, cache.size, len(items))
			})
		}
	}
}
