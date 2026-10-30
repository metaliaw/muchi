package search

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// CardService Answers what one Card Is, through the Providers of its Game: its
// Metadata, and the Names that Complete what a Person has Typed so far.
type CardService struct {
	Metadata     map[model.Game]cardmetadata.Provider
	Autocomplete map[model.Game]cardmetadata.AutocompleteProvider
	Names        map[model.Game]cardmetadata.NameProvider
	// Languages Names the Languages each Game Declares. A Game Absent from it
	// Declared none: any Language is Passed to its Provider as Written.
	Languages map[model.Game][]string
}

// checkLanguage Refuses a Language the Game does not Declare. A blank one is
// the Provider's own Default, and a Game with no Declaration Accepts them all.
func (s CardService) checkLanguage(game model.Game, language string) error {
	declared, found := s.Languages[game]
	if language == "" || !found || slices.Contains(declared, language) {
		return nil
	}
	return ErrUnsupportedLanguage
}

// ReadCardMetadata Asks the Game's Provider about one Card. A Game with no
// Provider Answers ErrNoCardMetadata, and a blank Name ErrNoCardName.
func (s CardService) ReadCardMetadata(ctx context.Context, game model.Game, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	provider, found := s.Metadata[game]
	if !found {
		return cardmetadata.Metadata{}, ErrNoCardMetadata
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		return cardmetadata.Metadata{}, ErrNoCardName
	}
	if err := s.checkLanguage(game, request.Language); err != nil {
		return cardmetadata.Metadata{}, err
	}
	return provider.CardMetadata(ctx, request)
}

// SuggestCardNames Completes a Card Name in one Language through the Game's
// Provider. A Game with no Provider Answers ErrNoAutocomplete.
func (s CardService) SuggestCardNames(ctx context.Context, game model.Game, name, language string) ([]string, error) {
	provider, found := s.Autocomplete[game]
	if !found {
		return nil, ErrNoAutocomplete
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrNoCardName
	}
	if err := s.checkLanguage(game, language); err != nil {
		return nil, err
	}
	return provider.Autocomplete(ctx, name, language)
}

// ResolveCardName Reads a Name Written in one Language as the Canonical Name,
// through the Game's Provider. A Game with no Provider Answers ErrNoCardNames,
// and a Name the Catalog does not Know ErrCardNotFound.
func (s CardService) ResolveCardName(ctx context.Context, game model.Game, name, language string) (string, error) {
	provider, found := s.Names[game]
	if !found {
		return "", ErrNoCardNames
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ErrNoCardName
	}
	if err := s.checkLanguage(game, language); err != nil {
		return "", err
	}
	canonical, err := provider.CanonicalName(ctx, name, language)
	if errors.Is(err, cardmetadata.ErrCardNotFound) {
		return "", ErrCardNotFound
	}
	return canonical, err
}
