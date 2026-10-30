package cardmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func (s Scryfall) CardMetadata(ctx context.Context, request Request) (Metadata, error) {
	return s.findArt(ctx, request)
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
