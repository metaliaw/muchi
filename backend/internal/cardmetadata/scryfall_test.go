package cardmetadata

import (
	"context"
	"strings"
	"testing"
)

type fakeFetcher struct {
	replies [][]byte
	targets []string
}

func (f *fakeFetcher) FetchSource(_ context.Context, _, target string) ([]byte, error) {
	f.targets = append(f.targets, target)
	reply := f.replies[0]
	f.replies = f.replies[1:]
	return reply, nil
}

func TestScryfallReturnsArt(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{
		[]byte(`{"data":[{"name":"Sol Ring","printed_name":"Anillo solar","scryfall_uri":"https://scryfall.com/card/x","image_uris":{"normal":"https://cards.scryfall.io/sol.jpg"}}]}`),
	}}
	provider := Scryfall{Fetcher: fetcher}

	metadata, err := provider.CardMetadata(context.Background(), Request{Name: "Anillo", Language: "es"})

	if err != nil || metadata.Image != "https://cards.scryfall.io/sol.jpg" {
		t.Fatalf("CardMetadata() metadata=%#v err=%v", metadata, err)
	}
}

func TestScryfallAutocompletesThreeUniqueNamesInLanguage(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{
		[]byte(`{"data":[{"name":"Sol Ring","printed_name":"Anillo solar"},{"name":"Sol Ring","printed_name":"Anillo solar"},{"name":"Sol Talisman"},{"name":"Solitude"},{"name":"Solar Tide"}]}`),
	}}
	provider := Scryfall{Fetcher: fetcher}

	suggestions, err := provider.Autocomplete(context.Background(), "Sol", "es")

	if err != nil || len(suggestions) != 3 {
		t.Fatalf("Autocomplete() suggestions=%#v err=%v", suggestions, err)
	}
	if suggestions[0] != "Anillo solar" || suggestions[1] != "Sol Talisman" || suggestions[2] != "Solitude" {
		t.Fatalf("suggestions=%#v", suggestions)
	}
	if !strings.Contains(fetcher.targets[0], "lang%3Aes") || !strings.Contains(fetcher.targets[0], "include_multilingual=true") {
		t.Fatalf("target=%q", fetcher.targets[0])
	}
}

func TestScryfallAsksForExactFoilPrinting(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{
		[]byte(`{"data":[{"name":"Sol Ring","image_uris":{"normal":"https://cards.scryfall.io/c21.jpg"}}]}`),
	}}
	provider := Scryfall{Fetcher: fetcher}

	metadata, err := provider.CardMetadata(context.Background(), Request{Name: "Sol Ring", Edition: "c21", Foil: true})

	if err != nil || metadata.Image == "" {
		t.Fatalf("CardMetadata() metadata=%#v err=%v", metadata, err)
	}
	if !strings.Contains(fetcher.targets[0], "%21%22Sol+Ring%22+set%3Ac21+is%3Afoil") {
		t.Fatalf("target=%q", fetcher.targets[0])
	}
}

func TestScryfallUsesTheFrontFaceImage(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{
		[]byte(`{"name":"Delver of Secrets","card_faces":[{"image_uris":{"normal":"https://cards.scryfall.io/front.jpg"}},{"image_uris":{"normal":"https://cards.scryfall.io/back.jpg"}}]}`),
	}}
	provider := Scryfall{Fetcher: fetcher}

	metadata, err := provider.CardMetadata(context.Background(), Request{Name: "Delver"})

	if err != nil || metadata.Image != "https://cards.scryfall.io/front.jpg" {
		t.Fatalf("CardMetadata() metadata=%#v err=%v", metadata, err)
	}
}
