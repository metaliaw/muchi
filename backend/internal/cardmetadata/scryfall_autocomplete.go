package cardmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const autocompleteLimit = 3

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
