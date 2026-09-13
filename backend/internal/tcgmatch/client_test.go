package tcgmatch

import (
	"context"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
)

type fixtureFetcher struct {
	targets []string
	catalog string
	listing string
}

func TestMetadataAndAutocompleteUseCatalogFields(t *testing.T) {
	catalog := `{"products":[{"id":1,"name":"Dark Magician","tcg":"yugioh","type":"card","image":"https://images.example/dark-magician.jpg","setCode":"LOB","languages":["english"]},{"id":2,"name":"Dark Magician Girl","tcg":"yugioh","type":"card","languages":["english"]},{"id":3,"name":"Dark Magician Knight","tcg":"yugioh","type":"card","languages":["english"]},{"id":4,"name":"Dark Magician Token","tcg":"yugioh","type":"card","languages":["english"]}]}`
	fetcher := &fixtureFetcher{catalog: catalog}
	client := Client{Fetcher: fetcher, BaseURL: "https://api.tcgmatch.cl", Game: "yugioh"}

	metadata, err := client.CardMetadata(context.Background(), cardmetadata.Request{Name: "Dark Magician", Language: "english", Edition: "LOB"})
	if err != nil || metadata.Name != "Dark Magician" || metadata.Edition != "LOB" || metadata.Image != "https://images.example/dark-magician.jpg" {
		t.Fatalf("CardMetadata() metadata=%#v err=%v", metadata, err)
	}
	suggestions, err := client.Autocomplete(context.Background(), "Dark", "english")
	if err != nil || len(suggestions) != 3 {
		t.Fatalf("Autocomplete() suggestions=%#v err=%v", suggestions, err)
	}
	if len(fetcher.targets) != 2 {
		t.Fatalf("catalog requests=%d", len(fetcher.targets))
	}
}

func (f *fixtureFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.targets = append(f.targets, target)
	if strings.Contains(target, "/catalog/search?") {
		return []byte(f.catalog), nil
	}
	return []byte(f.listing), nil
}

func TestSearch(t *testing.T) {
	tests := []struct {
		name, game, card, catalog, listing, price, language, finish string
	}{
		{
			name: "Pokemon", game: "pokemon", card: "Charizard ex", price: "1450", language: "spanish", finish: "holo",
			catalog: `{"products":[{"id":31969,"name":"Charizard ex - 125/197","tcg":"pokemon","type":"card","image":"https://images.example/charizard.jpg","setId":"sv03","setCode":"OBF"}]}`,
			listing: `{"success":true,"data":[{"_id":"listing-one","tcg":"pokemon","language":"spanish","status":"near-mint","quantity":1,"price":1450,"isActive":true,"isHolo":true,"user":{"name":"Cartas Uno","username":"cartas-uno"}}]}`,
		},
		{
			name: "YuGiOh", game: "yugioh", card: "Dark Magician", price: "3011", language: "english", finish: "nonfoil",
			catalog: `{"products":[{"id":208991,"name":"Dark Magician","tcg":"yugioh","type":"card"}]}`,
			listing: `{"success":true,"data":[{"_id":"listing-one","tcg":"yugioh","language":"english","status":"near-mint","quantity":1,"price":3011,"isActive":true,"user":{"name":"juan.brgsos","username":"juan.brgsos"}}]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fetcher := &fixtureFetcher{catalog: test.catalog, listing: test.listing}
			items, err := (Client{Fetcher: fetcher, BaseURL: "https://api.tcgmatch.cl", Game: test.game}).Search(context.Background(), test.card)
			if err != nil || len(items) != 1 {
				t.Fatalf("Search() items=%v err=%v", items, err)
			}
			item := items[0]
			if len(fetcher.targets) != 2 || !strings.Contains(fetcher.targets[0], "q="+strings.ReplaceAll(test.card, " ", "+")) || !strings.Contains(fetcher.targets[0], "tcg="+test.game) {
				t.Fatalf("targets=%v", fetcher.targets)
			}
			if item.ID != "tcgmatch:listing-one" || item.PriceAmount != test.price || item.Language != test.language || item.Finish != test.finish || item.Metadata["game"] != test.game {
				t.Fatalf("offer=%+v", item)
			}
			if test.game == "pokemon" && (item.Metadata["set_code"] != "OBF" || item.Metadata["set_id"] != "sv03" || item.Metadata["image"] == "") {
				t.Fatalf("pokemon metadata=%v", item.Metadata)
			}
		})
	}
}
