package stores

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/stores/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"
)

// SourceFetcher Reads Store pages interpreted with https://schema.org/availability.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// CommerceAdapter Checks Stock according to one Store Platform.
type CommerceAdapter interface {
	CheckStock(context.Context, offer.Offer) (offer.StockReading, error)
}

type Checker struct {
	Fetcher SourceFetcher
	Config  Config
	// Lists Answers for the Stores that Publish an Inventory instead of a
	// Storefront. Their Offers all Share one URL, so the Host below Names no
	// Store and the Question would Die as "unknown".
	Lists CommerceAdapter
}

func (c Checker) CheckStock(ctx context.Context, item offer.Offer) (offer.StockReading, error) {
	if c.Lists != nil && item.Source == "moxfield" {
		return c.Lists.CheckStock(ctx, item)
	}
	link, err := url.Parse(item.URL)
	if err != nil {
		return offer.ReadStock("unknown"), err
	}
	config, found := c.Config.Stores[link.Host]
	if !found || !config.Enabled {
		return offer.ReadStock("unknown"), nil
	}
	if adapter := c.commerceAdapter(config, link.Host); adapter != nil {
		return adapter.CheckStock(ctx, item)
	}
	data, err := c.Fetcher.FetchSource(ctx, link.Host, item.URL)
	if err != nil {
		return offer.ReadStock("unknown"), err
	}
	status := inspectStock(data, config)
	units := offer.CountDeclaredUnits(data)
	if units == nil {
		return offer.ReadStock(status), nil
	}
	// A Page that Counts its Units has Answered the Question: none Left is
	// Sold out, and any Number is a Store Saying yes with a Figure behind it.
	if *units == 0 {
		return offer.CountStock("unavailable", 0), nil
	}
	if status == "unavailable" {
		return offer.CountStock("unavailable", 0), nil
	}
	return offer.CountStock("available", *units), nil
}

func (c Checker) commerceAdapter(config StoreConfig, domain string) CommerceAdapter {
	switch config.Platform {
	case "jumpseller":
		return jumpseller.Client{Fetcher: c.Fetcher, Domain: domain, Name: config.Name}
	case "shopify":
		return shopify.Client{Fetcher: c.Fetcher, Domain: domain, Name: config.Name}
	default:
		return nil
	}
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
