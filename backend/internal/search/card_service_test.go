package search

import (
	"context"
	"errors"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type stubCardProvider struct{}

func (stubCardProvider) CardMetadata(_ context.Context, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	return cardmetadata.Metadata{Name: request.Name}, nil
}

func (stubCardProvider) Autocomplete(_ context.Context, name, _ string) ([]string, error) {
	return []string{name}, nil
}

func TestReadCardMetadataTrimsTheName(t *testing.T) {
	service := CardService{Metadata: map[model.Game]cardmetadata.Provider{model.GameMagic: stubCardProvider{}}}
	metadata, err := service.ReadCardMetadata(context.Background(), model.GameMagic, cardmetadata.Request{Name: "  Sol Ring "})
	if err != nil || metadata.Name != "Sol Ring" {
		t.Fatalf("ReadCardMetadata() metadata=%#v err=%v", metadata, err)
	}
}

func TestReadCardMetadataRefusesWhatItCannotAnswer(t *testing.T) {
	service := CardService{Metadata: map[model.Game]cardmetadata.Provider{model.GameMagic: stubCardProvider{}}}
	_, err := service.ReadCardMetadata(context.Background(), model.GamePokemon, cardmetadata.Request{Name: "Pikachu"})
	if !errors.Is(err, ErrNoCardMetadata) {
		t.Fatalf("game without a provider err=%v", err)
	}
	_, err = service.ReadCardMetadata(context.Background(), model.GameMagic, cardmetadata.Request{Name: "   "})
	if !errors.Is(err, ErrNoCardName) {
		t.Fatalf("blank name err=%v", err)
	}
}

func TestSuggestCardNamesRefusesWhatItCannotAnswer(t *testing.T) {
	service := CardService{Autocomplete: map[model.Game]cardmetadata.AutocompleteProvider{model.GameMagic: stubCardProvider{}}}
	names, err := service.SuggestCardNames(context.Background(), model.GameMagic, " Sol ", "english")
	if err != nil || len(names) != 1 || names[0] != "Sol" {
		t.Fatalf("SuggestCardNames() names=%#v err=%v", names, err)
	}
	_, err = service.SuggestCardNames(context.Background(), model.GamePokemon, "Pika", "english")
	if !errors.Is(err, ErrNoAutocomplete) {
		t.Fatalf("game without a provider err=%v", err)
	}
	_, err = service.SuggestCardNames(context.Background(), model.GameMagic, "", "english")
	if !errors.Is(err, ErrNoCardName) {
		t.Fatalf("blank name err=%v", err)
	}
}

// Un Juego que Declara sus Idiomas no Acepta otros: pasarle «ja» a un Catálogo
// que solo Anuncia Español e Inglés Contestaría vacío, y vacío Parece que la
// Carta no Existe.
func TestACardQuestionRefusesALanguageTheGameDoesNotDeclare(t *testing.T) {
	service := CardService{
		Metadata:     map[model.Game]cardmetadata.Provider{model.GameMagic: stubCardProvider{}},
		Autocomplete: map[model.Game]cardmetadata.AutocompleteProvider{model.GameMagic: stubCardProvider{}},
		Languages:    map[model.Game][]string{model.GameMagic: {"es", "en"}},
	}
	_, err := service.ReadCardMetadata(context.Background(), model.GameMagic, cardmetadata.Request{Name: "Sol Ring", Language: "ja"})
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("metadata in an undeclared language err=%v", err)
	}
	_, err = service.SuggestCardNames(context.Background(), model.GameMagic, "Sol", "ja")
	if !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("autocomplete in an undeclared language err=%v", err)
	}
	for _, language := range []string{"", "es", "en"} {
		if _, err := service.SuggestCardNames(context.Background(), model.GameMagic, "Sol", language); err != nil {
			t.Fatalf("language %q err=%v", language, err)
		}
	}
}

func TestAGameThatDeclaresNoLanguagesPassesThemAlong(t *testing.T) {
	service := CardService{
		Autocomplete: map[model.Game]cardmetadata.AutocompleteProvider{model.GamePokemon: stubCardProvider{}},
		Languages:    map[model.Game][]string{model.GameMagic: {"es", "en"}},
	}
	if _, err := service.SuggestCardNames(context.Background(), model.GamePokemon, "Pika", "ja"); err != nil {
		t.Fatalf("undeclared game err=%v", err)
	}
}

type stubNameProvider struct {
	canonical string
	failure   error
	asked     []string
}

func (s *stubNameProvider) CanonicalName(_ context.Context, name, language string) (string, error) {
	s.asked = append(s.asked, name+"|"+language)
	return s.canonical, s.failure
}

func TestResolveCardNameTrimsAndAsksTheGamesProvider(t *testing.T) {
	provider := &stubNameProvider{canonical: "Sol Ring"}
	service := CardService{Names: map[model.Game]cardmetadata.NameProvider{model.GameMagic: provider}}

	name, err := service.ResolveCardName(context.Background(), model.GameMagic, "  Anillo Solar ", "es")

	if err != nil || name != "Sol Ring" || len(provider.asked) != 1 || provider.asked[0] != "Anillo Solar|es" {
		t.Fatalf("ResolveCardName() name=%q err=%v asked=%v", name, err, provider.asked)
	}
}

func TestResolveCardNameRefusesWhatItCannotAnswer(t *testing.T) {
	provider := &stubNameProvider{canonical: "Sol Ring"}
	service := CardService{
		Names:     map[model.Game]cardmetadata.NameProvider{model.GameMagic: provider},
		Languages: map[model.Game][]string{model.GameMagic: {"es", "en"}},
	}
	if _, err := service.ResolveCardName(context.Background(), model.GamePokemon, "Pikachu", ""); !errors.Is(err, ErrNoCardNames) {
		t.Fatalf("game without a provider err=%v", err)
	}
	if _, err := service.ResolveCardName(context.Background(), model.GameMagic, "   ", ""); !errors.Is(err, ErrNoCardName) {
		t.Fatalf("blank name err=%v", err)
	}
	if _, err := service.ResolveCardName(context.Background(), model.GameMagic, "Sol Ring", "ja"); !errors.Is(err, ErrUnsupportedLanguage) {
		t.Fatalf("undeclared language err=%v", err)
	}
	if len(provider.asked) != 0 {
		t.Fatalf("the provider was asked anyway: %v", provider.asked)
	}
}

// El Proveedor Habla del Catálogo y el Servicio del Dominio: quien Llama atrapa
// ErrCardNotFound de aquí, jamás el del Adaptador.
func TestResolveCardNameAnswersTheDomainsNotFound(t *testing.T) {
	service := CardService{Names: map[model.Game]cardmetadata.NameProvider{
		model.GameMagic: &stubNameProvider{failure: cardmetadata.ErrCardNotFound},
	}}

	_, err := service.ResolveCardName(context.Background(), model.GameMagic, "zzzz", "")

	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("err=%v", err)
	}
}
