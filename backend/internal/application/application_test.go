package application

import (
	"slices"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/prestashop"
	"github.com/cangrejometralleta/muchi-api/internal/scry"
	"github.com/cangrejometralleta/muchi-api/internal/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

func TestConfiguredSources(t *testing.T) {
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	for domain, store := range config.Stores {
		store.Enabled = true
		config.Stores[domain] = store
	}
	sources := buildOfferSources(nil, scry.Client{}, config, nil, time.Minute, nil)
	lists := 0
	catalogs := 0
	shopifyStores := 0
	jumpsellerStores := 0
	prestashopStores := 0
	for _, source := range sources {
		switch value := source.(type) {
		case *moxfield.Client:
			lists++
			if value.Rate < 1 || value.Store == "" {
				t.Fatalf("list=%+v", value)
			}
		case jumpseller.Client:
			jumpsellerStores++
		case shopify.Client:
			shopifyStores++
		case prestashop.Client:
			prestashopStores++
		case stores.Catalog:
			catalogs++
		}
	}
	if lists != 9 || catalogs != 3 || shopifyStores != 4 || jumpsellerStores != 5 || prestashopStores != 1 || len(sources) != 23 {
		t.Fatalf("sources=%d lists=%d catalogs=%d", len(sources), lists, catalogs)
	}
	for _, domain := range []string{"gameofmagicsingles.cl", "singles.collectorcenter.cl", "www.cardsouls.cl", "www.oasisgames.cl"} {
		store, found := config.Stores[domain]
		if !found || !store.Enabled || store.Platform != "shopify" {
			t.Fatalf("Shopify store missing: %s", domain)
		}
	}
	store := config.Stores["el-wombat-rabioso-tcg"]
	store.Enabled = false
	config.Stores["el-wombat-rabioso-tcg"] = store
	if sources := buildOfferSources(nil, scry.Client{}, config, nil, time.Minute, nil); len(sources) != 14 {
		t.Fatalf("disabled lists still active: %d", len(sources))
	}
	for domain, store := range config.Stores {
		if store.Platform == "shopify" {
			store.Enabled = false
			config.Stores[domain] = store
		}
	}
	if sources := buildOfferSources(nil, scry.Client{}, config, nil, time.Minute, nil); len(sources) != 10 {
		t.Fatalf("disabled Shopify stores still active: %d", len(sources))
	}
	for domain, store := range config.Stores {
		if store.Platform == "jumpseller" {
			store.Enabled = false
			config.Stores[domain] = store
		}
	}
	if sources := buildOfferSources(nil, scry.Client{}, config, nil, time.Minute, nil); len(sources) != 5 {
		t.Fatalf("disabled Jumpseller stores still active: %d", len(sources))
	}
}

func TestConfiguredLocationNamesItsStore(t *testing.T) {
	config := stores.Config{Stores: map[string]stores.StoreConfig{
		"cards.test": {
			Name: "Cards Test",
			Locations: []stores.StoreLocation{
				{Country: "Chile", Region: "Valparaíso", City: "Viña del Mar"},
				{Country: "Chile", Region: "Valparaíso", City: "Quilpué", Pickup: "Mesa 1"},
			},
		},
		"nowhere.test": {},
	}}

	locations := buildStoreLocations(config)
	wanted := []string{"Viña del Mar", "Quilpué - Mesa 1"}
	if !slices.Equal(locations["cards.test"], wanted) || !slices.Equal(locations["Cards Test"], wanted) {
		t.Fatalf("locations=%v", locations)
	}
	if _, found := locations["nowhere.test"]; found {
		t.Fatalf("empty location published: %v", locations)
	}
}

func TestSourcesWithoutScry(t *testing.T) {
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	sources := buildOfferSources(nil, nil, config, nil, time.Minute, nil)
	for _, source := range sources {
		if _, found := source.(scry.Client); found {
			t.Fatal("Scry still active")
		}
	}
	if len(sources) != 18 {
		t.Fatalf("sources=%d", len(sources))
	}
}

func TestSlowCatalogsPausedWithScryAvailable(t *testing.T) {
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, domain := range []string{"www.magic4ever.cl", "www.cartaslafortaleza.cl", "www.chronomagic.cl", "gamequest.cl"} {
		store, found := config.Stores[domain]
		if !found || store.Enabled {
			t.Fatalf("slow catalog active or missing: %s", domain)
		}
	}
	sources := buildOfferSources(nil, scry.Client{}, config, nil, time.Minute, nil)
	if len(sources) != 19 {
		t.Fatalf("sources=%d", len(sources))
	}
	if _, ok := sources[0].(scry.Client); !ok {
		t.Fatal("Scry missing")
	}
	for _, source := range sources {
		if client, ok := source.(jumpseller.Client); ok && client.Domain != "www.deckscards.cl" {
			t.Fatal("slow catalog still active")
		}
	}
}
