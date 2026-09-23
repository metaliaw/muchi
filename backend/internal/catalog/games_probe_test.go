package catalog

import (
	"context"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

// A Game that Loses its Sources Keeps answering, just Empty. Un Probe por Juego
// Muestra cuál Dejó de Traer Ofertas antes de que la Búsqueda lo Calle.
func TestEveryGameProbe(t *testing.T) {
	if os.Getenv("MUCHI_LIVE") == "" {
		t.Skip("set MUCHI_LIVE=1")
	}
	config, err := stores.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := probeFetcher()
	service := search.Service{
		Providers:     BuildProviders(fetcher, config),
		SourcesByGame: BuildSourcesByGame(fetcher, config, nil, time.Hour, nil),
	}
	for _, probe := range []struct {
		game search.Game
		card string
	}{
		{search.GameMagic, "Sol Ring"},
		{search.GamePokemon, "Pikachu"},
		{search.GameYuGiOh, "Dark Magician"},
		{search.GameOnePiece, "Monkey D. Luffy"},
		{search.GameDigimon, "Agumon"},
		{search.GameRiftbound, "Jinx"},
		{search.GameMitos, "Dragón de Magma"},
	} {
		t.Run(string(probe.game), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			started := time.Now()
			items, faults, err := service.FindCardOffers(ctx, probe.game, offer.CardQuery{Name: probe.card})
			if err != nil {
				t.Fatal(err)
			}
			for _, fault := range faults {
				t.Logf("fault source=%s reason=%s", fault.Source, fault.Reason)
			}
			bySource := map[string]int{}
			byStore := map[string]int{}
			origins := map[string]map[string]bool{}
			seen := map[string]bool{}
			for _, item := range items {
				bySource[item.Source]++
				byStore[item.Store]++
				if product := item.Metadata["product_id"]; !seen[product] {
					seen[product] = true
					t.Logf("  card=%q card_key=%q set_id=%q set_code=%q",
						item.CardName, item.CardKey, item.Metadata["set_id"], item.Metadata["set_code"])
				}
				key := sameOffer(item)
				if origins[key] == nil {
					origins[key] = map[string]bool{}
				}
				origins[key][item.Source] = true
			}
			duplicates := 0
			for key, sources := range origins {
				if len(sources) > 1 {
					duplicates++
					t.Logf("  duplicate across sources=%s sources=%d", key, len(sources))
				}
			}
			t.Logf("game=%s card=%q duration=%s offers=%d sources=%d stores=%d duplicates=%d",
				probe.game, probe.card, time.Since(started), len(items), len(bySource), len(byStore), duplicates)
			for _, name := range sortedKeys(bySource) {
				t.Logf("  source=%s offers=%d", name, bySource[name])
			}
			if len(items) == 0 {
				t.Fatalf("%s returned zero offers for %q", probe.game, probe.card)
			}
		})
	}
}

func sortedKeys(counts map[string]int) []string {
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sameOffer Names the Offer a Reader would Call the same one, whichever Source
// Brought it. Dos Fuentes que Traen la misma Carta al mismo Precio Cuentan una.
func sameOffer(item offer.Offer) string {
	return strings.Join([]string{
		offer.NormalizeCard(item.CardName), strings.ToLower(item.Store),
		item.PriceAmount, item.PriceCurrency, strings.ToLower(item.Language),
	}, "|")
}
