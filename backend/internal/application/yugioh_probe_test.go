package application

import (
	"context"
	"fmt"
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

func TestYuGiOhDarkMagicianProbe(t *testing.T) {
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
	items, err := service.FindCardOffers(ctx, search.GameYuGiOh, "Dark Magician")
	if err != nil {
		t.Fatal(err)
	}
	bySource := map[string]int{}
	semantic := map[string][]offer.Offer{}
	for _, item := range items {
		bySource[item.Source]++
		key := strings.Join([]string{offer.NormalizeCard(item.CardName), strings.ToLower(item.Store), item.PriceAmount, item.PriceCurrency, strings.ToLower(item.Language)}, "|")
		semantic[key] = append(semantic[key], item)
	}
	sources := make([]string, 0, len(bySource))
	for name := range bySource {
		sources = append(sources, name)
	}
	sort.Strings(sources)
	duplicates := 0
	for key, group := range semantic {
		origins := map[string]bool{}
		for _, item := range group {
			origins[item.Source] = true
		}
		if len(origins) > 1 {
			duplicates += len(group) - 1
			t.Logf("cross-source duplicate=%s offers=%d", key, len(group))
		}
	}
	t.Logf("duration=%s offers=%d cross_source_duplicates=%d", time.Since(started), len(items), duplicates)
	for _, name := range sources {
		t.Logf("source=%s offers=%d", name, bySource[name])
	}
	if len(items) == 0 {
		t.Fatal(fmt.Errorf("combined search returned no offers"))
	}
}