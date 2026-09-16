package application

import (
	"context"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

func TestPokemonSlowpokeProbe(t *testing.T) {
	if os.Getenv("MUCHI_LIVE") == "" {
		t.Skip("set MUCHI_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := source.Client{
		HTTP: &http.Client{Timeout: 20 * time.Second}, Gate: openGate{},
		UserAgent: userAgent, MaxAttempts: 2, BaseDelay: 250 * time.Millisecond, MaxBodyBytes: 4 << 20,
	}
	service := search.Service{
		Providers:     buildProviders(fetcher, config),
		SourcesByGame: buildSourcesByGame(fetcher, config, nil, time.Hour, nil),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	started := time.Now()
	items, _, err := service.FindCardOffers(ctx, search.GamePokemon, offer.CardQuery{Name: "Slowpoke"})
	if err != nil {
		t.Fatal(err)
	}
	bySource := map[string]int{}
	semantic := map[string]map[string]bool{}
	functions := map[string]bool{}
	products := map[string]bool{}
	for _, item := range items {
		bySource[item.Source]++
		functions[item.CardKey] = true
		product := item.Metadata["product_id"]
		if !products[product] {
			t.Logf("card=%q card_key=%q set_id=%q set_code=%q", item.CardName, item.CardKey, item.Metadata["set_id"], item.Metadata["set_code"])
			products[product] = true
		}
		key := strings.Join([]string{offer.NormalizeCard(item.CardName), strings.ToLower(item.Store), item.PriceAmount, item.PriceCurrency, strings.ToLower(item.Language)}, "|")
		if semantic[key] == nil {
			semantic[key] = map[string]bool{}
		}
		semantic[key][item.Source] = true
	}
	duplicates := 0
	for key, origins := range semantic {
		if len(origins) > 1 {
			duplicates++
			t.Logf("cross-source duplicate=%s sources=%d", key, len(origins))
		}
	}
	sources := make([]string, 0, len(bySource))
	for name := range bySource {
		sources = append(sources, name)
	}
	sort.Strings(sources)
	t.Logf("duration=%s offers=%d functions=%d cross_source_duplicates=%d", time.Since(started), len(items), len(functions), duplicates)
	for _, name := range sources {
		t.Logf("source=%s offers=%d", name, bySource[name])
	}
}
