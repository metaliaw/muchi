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

// NameProvider Reads a Card Name Written in any Language as the Canonical one.
// A Name the Catalog does not Know Answers ErrCardNotFound.
type NameProvider interface {
	CanonicalName(ctx context.Context, name, language string) (string, error)
}

// PrintProvider Lists the known Printings of one Card.
type PrintProvider interface {
	CardPrints(context.Context, string) ([]Print, error)
}
