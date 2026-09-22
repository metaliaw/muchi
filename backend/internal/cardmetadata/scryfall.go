package cardmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const scryfallDomain = "api.scryfall.com"
const autocompleteLimit = 3

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

func (s Scryfall) CardMetadata(ctx context.Context, request Request) (Metadata, error) {
	return s.findArt(ctx, request)
}

func (s Scryfall) CardPrints(ctx context.Context, name string) ([]Print, error) {
	query := url.Values{
		"q":      {fmt.Sprintf(`!%q`, name)},
		"unique": {"prints"},
	}
	body, err := s.fetch(ctx, "/cards/search", query)
	if err != nil {
		return nil, err
	}
	var result cardList
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	prints := make([]Print, 0, len(result.Data))
	for index := range result.Data {
		metadata := metadataOf(&result.Data[index])
		if metadata.Image == "" {
			continue
		}
		prints = append(prints, Print{
			Edition:         strings.ToLower(result.Data[index].Set),
			EditionName:     strings.ToLower(result.Data[index].SetName),
			CollectorNumber: strings.ToLower(result.Data[index].CollectorNumber),
			Image:           metadata.Image,
		})
	}
	return prints, nil
}

func (s Scryfall) findArt(ctx context.Context, request Request) (Metadata, error) {
	if request.Edition != "" {
		query := fmt.Sprintf(`!%q set:%s`, request.Name, request.Edition)
		if request.Foil {
			query += " is:foil"
		}
		card, err := s.onlyCard(ctx, "/cards/search", url.Values{"q": {query}, "unique": {"prints"}})
		if err == nil && card != nil {
			return metadataOf(card), nil
		}
	}
	if request.Language != "" {
		return s.artFromSearch(ctx, request)
	}
	card, err := s.firstCard(ctx, "/cards/named", url.Values{"fuzzy": {request.Name}})
	if err != nil || card == nil {
		return Metadata{}, err
	}
	return metadataOf(card), nil
}

func (s Scryfall) onlyCard(ctx context.Context, path string, query url.Values) (*scryfallCard, error) {
	body, err := s.fetch(ctx, path, query)
	if err != nil {
		return nil, err
	}
	var result cardList
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data) != 1 {
		return nil, err
	}
	return &result.Data[0], nil
}

func (s Scryfall) artFromSearch(ctx context.Context, request Request) (Metadata, error) {
	card, err := s.firstCard(ctx, "/cards/search", url.Values{
		"q":                    {fmt.Sprintf(`lang:%s %q`, request.Language, request.Name)},
		"include_multilingual": {"true"},
		"unique":               {"cards"},
	})
	if err != nil || card == nil {
		return Metadata{}, err
	}
	return metadataOf(card), nil
}

func (s Scryfall) Autocomplete(ctx context.Context, text, language string) ([]string, error) {
	if text == "" {
		return []string{}, nil
	}
	// Sin Idioma no hay Consulta que Hacer, y Contestar vacío Parecía que la
	// Carta no Existe. El Catálogo Está escrito en Inglés: ese es el Idioma de
	// quien no Pidió otro.
	if language == "" {
		language = defaultLanguage
	}
	body, err := s.fetch(ctx, "/cards/search", url.Values{
		"q":                    {fmt.Sprintf(`lang:%s %q`, language, text)},
		"include_multilingual": {"true"},
		"unique":               {"cards"},
	})
	if err != nil {
		return nil, err
	}
	var result cardList
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	names := make([]string, 0, autocompleteLimit)
	seen := map[string]bool{}
	for _, card := range result.Data {
		name := card.PrintedName
		if name == "" {
			name = card.Name
		}
		if name != "" && !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
		if len(names) == autocompleteLimit {
			break
		}
	}
	return names, nil
}

func (s Scryfall) firstCard(ctx context.Context, path string, query url.Values) (*scryfallCard, error) {
	body, err := s.fetch(ctx, path, query)
	if err != nil {
		return nil, err
	}
	if path == "/cards/search" {
		var result cardList
		if err := json.Unmarshal(body, &result); err != nil || len(result.Data) == 0 {
			return nil, err
		}
		return &result.Data[0], nil
	}
	var card scryfallCard
	if err := json.Unmarshal(body, &card); err != nil {
		return nil, err
	}
	return &card, nil
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
