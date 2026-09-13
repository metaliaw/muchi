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

type Request struct {
	Name     string
	Language string
	Edition  string
	Foil     bool
}

type Metadata struct {
	Name        string `json:"name"`
	PrintedName string `json:"printed_name,omitempty"`
	Edition     string `json:"edition,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
}
