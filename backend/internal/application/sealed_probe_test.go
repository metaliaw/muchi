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

// TestSealedProbe Asks the real Sources for Sealed Product, to Learn whether a
// Store Sells more than Singles. It Reads, it never Writes.
func TestSealedProbe(t *testing.T) {
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
		SetsByGame:    buildSetLibraries(fetcher, config),
	}
	probes := []struct {
		game search.Game
		name string
	}{
		{search.GamePokemon, "Elite Trainer Box"},
		{search.GameYuGiOh, "Booster Box"},
		{search.GameMagic, "Collector Booster"},
		{search.GameYuGiOh, "Chaos Origins Booster Box"},
	}
	for _, probe := range probes {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		items, faults, err := service.FindCardOffers(ctx, probe.game,
			offer.CardQuery{Name: probe.name, Kind: offer.KindSealed})
		cancel()
		if err != nil {
			t.Logf("game=%s name=%q err=%v", probe.game, probe.name, err)
			continue
		}
		bySource := map[string]int{}
		titles := map[string]string{}
		for _, item := range items {
			bySource[item.Source]++
			flag := ""
			if item.Suspicious {
				flag = " SUSPICIOUS"
			}
			titles[item.Source+" | "+item.CardName] = item.PriceAmount + flag
		}
		t.Logf("game=%s name=%q offers=%d faults=%v", probe.game, probe.name, len(items), faults)
		keys := make([]string, 0, len(titles))
		for key := range titles {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			t.Logf("   %s -> %s", key, titles[key])
		}
	}
}
