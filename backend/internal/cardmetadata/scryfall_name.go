package cardmetadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// ErrCardNotFound Says the Catalog Knows no Card by that Name, in that Language
// or in any other. It is an Answer, not a Fault: nobody Needs to Retry it.
var ErrCardNotFound = errors.New("card not found")

// CanonicalName Reads a Card Name Written in any Language as the Name the Shops
// Index it under.
//
// Two Doors. /cards/named with fuzzy Guesses the Language alone and Reaches
// almost Always. /cards/search with lang: Narrows to one Language when a Name
// Exists in several and fuzzy Picks wrong; named Ignores lang in Silence, which
// is why a Language Changes the Door.
func (s Scryfall) CanonicalName(ctx context.Context, name, language string) (string, error) {
	if language == "" {
		return s.canonicalNamed(ctx, name)
	}
	return s.canonicalInLanguage(ctx, name, language)
}

func (s Scryfall) canonicalNamed(ctx context.Context, name string) (string, error) {
	body, err := s.fetch(ctx, "/cards/named", url.Values{"fuzzy": {name}})
	if err != nil {
		return "", missingAsNotFound(err)
	}
	var card scryfallCard
	if err := json.Unmarshal(body, &card); err != nil || card.Name == "" {
		return "", ErrCardNotFound
	}
	return card.Name, nil
}

func (s Scryfall) canonicalInLanguage(ctx context.Context, name, language string) (string, error) {
	// The Quotes Ask for the whole Phrase: without them every Word Searches alone.
	body, err := s.fetch(ctx, "/cards/search", url.Values{
		"q":                    {fmt.Sprintf(`lang:%s %q`, language, name)},
		"include_multilingual": {"true"},
		"order":                {"released"},
	})
	if err != nil {
		return "", missingAsNotFound(err)
	}
	var result cardList
	if err := json.Unmarshal(body, &result); err != nil || len(result.Data) == 0 || result.Data[0].Name == "" {
		return "", ErrCardNotFound
	}
	return result.Data[0].Name, nil
}

// missingAsNotFound Reads Scryfall's 404 as what it Means. Any other Failure
// Travels as it Came, so the Caller can Tell a Card that Does not Exist from a
// Catalog that Did not Answer.
func missingAsNotFound(err error) error {
	var status source.StatusError
	if errors.As(err, &status) && status.Code == http.StatusNotFound {
		return ErrCardNotFound
	}
	return err
}
