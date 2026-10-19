package tcgmatch

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher           SourceFetcher
	BaseURL           string
	Game              string
	PokemonCatalogURL string
}

func (c Client) validBase(text string) (*url.URL, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid TCGMatch URL")
	}
	if strings.TrimSpace(text) == "" || c.Game == "" {
		return nil, errors.New("invalid TCGMatch search")
	}
	return base, nil
}

// SourceName Identifies this Provider the Way the Health Report Names it.
func (c Client) SourceName() string {
	if base, err := url.Parse(c.BaseURL); err == nil && base.Host != "" {
		return base.Host
	}
	return c.BaseURL
}
