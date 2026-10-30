package cardmetadata

import (
	"context"
	"net/url"
)

const scryfallDomain = "api.scryfall.com"

// defaultLanguage Nombra el Idioma en que Scryfall Escribe una Carta cuando
// nadie Pide otro.
const defaultLanguage = "en"

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// Scryfall Implements Card Metadata using Scryfall's public card Catalog.
type Scryfall struct {
	Fetcher SourceFetcher
	BaseURL string
}

func (s Scryfall) fetch(ctx context.Context, path string, query url.Values) ([]byte, error) {
	base := s.BaseURL
	if base == "" {
		base = "https://api.scryfall.com"
	}
	return s.Fetcher.FetchSource(ctx, scryfallDomain, base+path+"?"+query.Encode())
}

type cardList struct {
	Data []scryfallCard `json:"data"`
}

type scryfallCard struct {
	Name            string            `json:"name"`
	PrintedName     string            `json:"printed_name"`
	Set             string            `json:"set"`
	SetName         string            `json:"set_name"`
	CollectorNumber string            `json:"collector_number"`
	ScryfallURL     string            `json:"scryfall_uri"`
	ImageURIs       map[string]string `json:"image_uris"`
	CardFaces       []scryfallCard    `json:"card_faces"`
}

func metadataOf(card *scryfallCard) Metadata {
	images := card.ImageURIs
	if len(images) == 0 && len(card.CardFaces) > 0 {
		images = card.CardFaces[0].ImageURIs
	}
	image := images["normal"]
	if image == "" {
		image = images["large"]
	}
	if image == "" {
		image = images["small"]
	}
	return Metadata{Name: card.Name, PrintedName: card.PrintedName, Edition: card.Set, Image: image, URL: card.ScryfallURL}
}
