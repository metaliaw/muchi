package catalog

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores/commerce"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

// TestImageCoverageProbe Counts how many real Offers Arrive with a Picture.
func TestImageCoverageProbe(t *testing.T) {
	if os.Getenv("MUCHI_LIVE") == "" {
		t.Skip("set MUCHI_LIVE=1")
	}
	config, err := directory.LoadStoreConfig("../../config/stores.yaml", nil)
	if err != nil {
		t.Fatal(err)
	}
	fetcher := probeFetcher()
	builder := Builder{Shops: commerce.Shops{Config: config, Fetcher: fetcher}, TTL: time.Hour}
	service := search.Service{
		OfferService: search.OfferService{
			Providers:     builder.Providers(),
			SourcesByGame: builder.SourcesByGame(),
			PrintsByGame:  builder.PrintLibraries(),
		},
	}
	for _, card := range []string{"Sol Ring", "Lightning Bolt"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		items, faults, err := service.FindCardOffers(ctx, model.GameMagic, model.CardQuery{Name: card})
		cancel()
		if err != nil {
			t.Fatalf("card=%q err=%v", card, err)
		}
		reportImages(t, card, items, faults)
	}
}

func reportImages(t *testing.T, card string, items []model.Offer, faults []model.SourceFault) {
	type tally struct{ offers, images int }
	bySource := map[string]*tally{}
	total := tally{}
	for _, item := range items {
		row, found := bySource[item.Source]
		if !found {
			row = &tally{}
			bySource[item.Source] = row
		}
		row.offers++
		total.offers++
		if item.Image != "" {
			row.images++
			total.images++
		}
	}
	names := make([]string, 0, len(bySource))
	for name := range bySource {
		names = append(names, name)
	}
	sort.Strings(names)
	t.Logf("== %s == offers=%d with image=%d (%d%%)", card, total.offers, total.images, percent(total.images, total.offers))
	for _, name := range names {
		row := bySource[name]
		t.Logf("   %-28s %3d offers  %3d with image  %d%%", name, row.offers, row.images, percent(row.images, row.offers))
	}
	for _, fault := range faults {
		t.Logf("   fault %s", fault.Source)
	}
}

func percent(part, whole int) int {
	if whole == 0 {
		return 0
	}
	return 100 * part / whole
}
