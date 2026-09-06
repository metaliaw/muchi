package stores

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// SourceFetcher Reads Store pages interpreted with https://schema.org/availability.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Checker struct {
	Fetcher SourceFetcher
	Config  Config
}

func (c Checker) CheckStock(ctx context.Context, item offer.Offer) (string, error) {
	link, err := url.Parse(item.URL)
	if err != nil {
		return "unknown", err
	}
	config, found := c.Config.Stores[link.Host]
	if !found || !config.Enabled {
		return "unknown", nil
	}
	data, err := c.Fetcher.FetchSource(ctx, link.Host, item.URL)
	if err != nil {
		return "unknown", err
	}
	return inspectStock(data, config), nil
}

func inspectStock(data []byte, config StoreConfig) string {
	text := strings.ToLower(string(data))
	for _, value := range config.UnavailableText {
		if strings.Contains(text, strings.ToLower(value)) {
			return "unavailable"
		}
	}
	for _, selector := range config.UnavailableSelectors {
		marker := strings.TrimPrefix(strings.ReplaceAll(selector, ".", " "), "#")
		if containsTerms(text, strings.Fields(marker)) {
			return "unavailable"
		}
	}
	if strings.Contains(text, "outofstock") || strings.Contains(text, "out-of-stock") {
		return "unavailable"
	}
	if inspectJSONLD(data) {
		return "available"
	}
	return "unknown"
}

func containsTerms(text string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(text, strings.ToLower(term)) {
			return false
		}
	}
	return len(terms) > 0
}

func inspectJSONLD(data []byte) bool {
	text := string(data)
	start := strings.Index(text, `{`)
	end := strings.LastIndex(text, `}`)
	if start < 0 || end <= start {
		return false
	}
	var value any
	if json.Unmarshal([]byte(text[start:end+1]), &value) != nil {
		return false
	}
	return strings.Contains(strings.ToLower(text[start:end+1]), "instock")
}

var ErrStoreUnsupported = errors.New("store is not configured")
