package application

import (
	"context"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

// TestMatchModeProbe Reads the same Card under both Modes against real Sources.
func TestMatchModeProbe(t *testing.T) {
	if os.Getenv("MUCHI_LIVE") == "" {
		t.Skip("set MUCHI_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := source.Client{
		HTTP: &http.Client{Timeout: 20 * time.Second}, Gate: openGate{},
		UserAgent: userAgent, MaxAttempts: 2, BaseDelay: 250 * time.Millisecond, MaxBodyBytes: 16 << 20,
	}
	service := search.Service{
		Providers:     buildProviders(fetcher, config),
		SourcesByGame: buildSourcesByGame(fetcher, config, nil, time.Hour, nil),
	}
	for _, mode := range []offer.MatchMode{offer.MatchExact, offer.MatchIncludes} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		items, _, err := service.FindCardOffers(ctx, search.GameYuGiOh, offer.CardQuery{Name: "Kuriboh", Match: mode})
		cancel()
		if err != nil {
			t.Fatalf("mode=%s err=%v", mode, err)
		}
		names := map[string]int{}
		for _, item := range items {
			names[item.CardName]++
		}
		titles := make([]string, 0, len(names))
		for name := range names {
			titles = append(titles, name)
		}
		sort.Strings(titles)
		t.Logf("mode=%s offers=%d distinct=%d", mode, len(items), len(titles))
		for _, title := range titles {
			t.Logf("   %2dx %s", names[title], title)
		}
	}
}
