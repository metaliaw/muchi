package application

import (
	"context"
	"os"
	"sort"
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
		Providers:     buildProviders(fetcher, config),
		SourcesByGame: buildSourcesByGame(fetcher, config, nil, time.Hour, nil),
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
				t.Logf("falla fuente=%s motivo=%s", fault.Source, fault.Reason)
			}
			bySource := map[string]int{}
			byStore := map[string]int{}
			for _, item := range items {
				bySource[item.Source]++
				byStore[item.Store]++
			}
			t.Logf("juego=%s carta=%q duracion=%s ofertas=%d fuentes=%d tiendas=%d",
				probe.game, probe.card, time.Since(started), len(items), len(bySource), len(byStore))
			for _, name := range sortedKeys(bySource) {
				t.Logf("  fuente=%s ofertas=%d", name, bySource[name])
			}
			if len(items) == 0 {
				t.Fatalf("%s devolvió cero ofertas para %q", probe.game, probe.card)
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
