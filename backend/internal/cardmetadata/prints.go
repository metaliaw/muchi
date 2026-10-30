package cardmetadata

import (
	"regexp"
	"strings"
)

// PrintIndex Answers which Printing a Store Title Names, for one Card.
// Indexed once, Asked by every Offer of that Card.
type PrintIndex struct {
	exact    map[string]string
	editions map[string]string
}

// Empty Says the Card has no usable Printing, so Asking is not worth a Loop.
func (p PrintIndex) Empty() bool { return len(p.editions) == 0 }

// IndexPrints Reads every Printing once so many Offers can Ask the same Question.
func IndexPrints(prints []Print) PrintIndex {
	exact := make(map[string]string, len(prints))
	editions := make(map[string]string)
	for _, print := range prints {
		number := strings.ToLower(print.CollectorNumber)
		if number == "" || print.Image == "" {
			continue
		}
		// Una Tienda Escribe `[C21]` y otra `[Fallout]`: el Código y el
		// Nombre Nombran la misma Edición, y Scryfall Trae los dos.
		// Indexar ambos Deja que el Título Elija el Idioma que Prefiera.
		for _, edition := range []string{strings.ToLower(print.Edition), strings.ToLower(print.EditionName)} {
			if edition == "" {
				continue
			}
			exact[edition+":"+number] = print.Image
			// La primera Impresión que Scryfall Lista Representa a su
			// Edición cuando la Oferta Calla el Número. Una Hermana
			// Muestra la Carta; un Hueco no Muestra nada.
			if _, seen := editions[edition]; !seen {
				editions[edition] = print.Image
			}
		}
	}
	return PrintIndex{exact: exact, editions: editions}
}

// ImageFor Finds the Image for whatever Shape a Store Chose to Write.
// Every Store Writes its own: `[C21] #263`, `| C20`, `(C19 #221)`,
// `[C14-270]`, `[MKC - 237]`, `- 214 -`. Guessing the Shape Needs one Rule
// per Store; Asking instead which Word *is* an Edition Scryfall Listed for
// this Card Needs none, and Never Invents one.
func (p PrintIndex) ImageFor(title, name string) string {
	words := titleWords(strings.TrimPrefix(strings.ToLower(title), strings.ToLower(name)))
	edition := longestEdition(words, p.editions)
	if edition == "" {
		return ""
	}
	// Every numeric Word is a Candidate Number, and a wrong Guess Costs
	// nothing: only a Pair the Print List Confirms Wins. When none does,
	// the Edition alone Answers with its first Impresión.
	for _, word := range words {
		if image := p.exact[edition+":"+word]; image != "" {
			return image
		}
	}
	return p.editions[edition]
}

var wordPattern = regexp.MustCompile(`[\p{L}\p{N}★]+`)

func titleWords(title string) []string {
	return wordPattern.FindAllString(title, -1)
}

// longestEdition Prefers `modern horizons 2` over a bare `2`, so a Name of
// several Words Wins against a Fragment of itself.
func longestEdition(words []string, editions map[string]string) string {
	const longestEditionName = 6
	for length := longestEditionName; length >= 1; length-- {
		for start := 0; start+length <= len(words); start++ {
			candidate := strings.Join(words[start:start+length], " ")
			if _, known := editions[candidate]; known {
				return candidate
			}
		}
	}
	return ""
}
