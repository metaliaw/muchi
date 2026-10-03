package cardmetadata

import "context"

// Provider Resolves presentation Metadata without exposing its external Source.
type Provider interface {
	CardMetadata(context.Context, Request) (Metadata, error)
}

// AutocompleteProvider Suggests card Names written in one Language.
type AutocompleteProvider interface {
	Autocomplete(context.Context, string, string) ([]string, error)
}

// PrintProvider Lists the known Printings of one Card.
type PrintProvider interface {
	CardPrints(context.Context, string) ([]Print, error)
}
