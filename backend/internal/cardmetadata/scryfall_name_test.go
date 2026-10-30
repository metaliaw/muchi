package cardmetadata

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// failingFetcher Answers every Request with the Failure it was Given.
type failingFetcher struct{ failure error }

func (f failingFetcher) FetchSource(context.Context, string, string) ([]byte, error) {
	return nil, f.failure
}

func TestCanonicalNameGuessesTheLanguageWhenNoneIsChosen(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{[]byte(`{"name":"Sol Ring"}`)}}

	name, err := Scryfall{Fetcher: fetcher}.CanonicalName(context.Background(), "Anillo Solar", "")

	if err != nil || name != "Sol Ring" {
		t.Fatalf("CanonicalName() name=%q err=%v", name, err)
	}
	if !strings.Contains(fetcher.targets[0], "/cards/named?") || !strings.Contains(fetcher.targets[0], "fuzzy=Anillo+Solar") {
		t.Fatalf("target=%q", fetcher.targets[0])
	}
}

// named Ignora lang en Silencio: por eso un Idioma Cambia de Puerta.
func TestCanonicalNameSwitchesToTheSearchDoorWithALanguage(t *testing.T) {
	fetcher := &fakeFetcher{replies: [][]byte{[]byte(`{"data":[{"name":"Sol Ring"},{"name":"Otra"}]}`)}}

	name, err := Scryfall{Fetcher: fetcher}.CanonicalName(context.Background(), "Anillo Solar", "es")

	if err != nil || name != "Sol Ring" {
		t.Fatalf("CanonicalName() name=%q err=%v", name, err)
	}
	target := fetcher.targets[0]
	// Las Comillas piden la Frase entera: sin ellas cada Palabra busca sola.
	if !strings.Contains(target, "/cards/search?") || !strings.Contains(target, "lang%3Aes+%22Anillo+Solar%22") ||
		!strings.Contains(target, "include_multilingual=true") || !strings.Contains(target, "order=released") {
		t.Fatalf("target=%q", target)
	}
}

func TestCanonicalNameReportsACardTheCatalogDoesNotKnow(t *testing.T) {
	missing := source.StatusError{Code: http.StatusNotFound}
	for name, fetcher := range map[string]SourceFetcher{
		"a 404 from the fuzzy door": failingFetcher{missing},
		"an empty search":           &fakeFetcher{replies: [][]byte{[]byte(`{"data":[]}`)}},
		"a nameless answer":         &fakeFetcher{replies: [][]byte{[]byte(`{}`)}},
	} {
		language := ""
		if strings.Contains(name, "search") {
			language = "ja"
		}
		_, err := Scryfall{Fetcher: fetcher}.CanonicalName(context.Background(), "Anillo Solar", language)
		if !errors.Is(err, ErrCardNotFound) {
			t.Errorf("%s err=%v", name, err)
		}
	}
	_, err := Scryfall{Fetcher: failingFetcher{missing}}.CanonicalName(context.Background(), "Anillo Solar", "es")
	if !errors.Is(err, ErrCardNotFound) {
		t.Errorf("a 404 from the search door err=%v", err)
	}
}

// Un Catálogo que no Contestó no es una Carta que no Existe: quien Pregunta
// debe poder Distinguirlos para Reintentar solo uno.
func TestCanonicalNameKeepsACatalogFailureApartFromANotFound(t *testing.T) {
	down := source.StatusError{Code: http.StatusServiceUnavailable}

	_, err := Scryfall{Fetcher: failingFetcher{down}}.CanonicalName(context.Background(), "Anillo Solar", "")

	if errors.Is(err, ErrCardNotFound) || !errors.Is(err, down) {
		t.Fatalf("err=%v", err)
	}
}
