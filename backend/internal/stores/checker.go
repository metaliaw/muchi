package stores

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/shopify"
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
}

func (c Checker) CheckStock(ctx context.Context, item offer.Offer) (offer.StockReading, error) {
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
	units := countUnits(data)
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

// declaredUnits Matches the Way a Storefront Writes what it has Left, in the
// Sentence a Buyer Reads: "3 disponibles", "1 unidad", "2 en stock".
var declaredUnits = regexp.MustCompile(`(?i)(\d{1,4})\s*(?:unidades?|disponibles?|en stock)`)

// countUnits Reads how many Units the Page Declares, or nil when it Stays
// quiet. Most Storefronts never Say a Number, and Guessing one would Turn a
// Silence into a Promise. The First Match Wins: it Sits next to the Quantity
// Box, before the Footer and its Unrelated Digits.
func countUnits(data []byte) *int {
	match := declaredUnits.FindSubmatch(data)
	if match == nil {
		return nil
	}
	units, err := strconv.Atoi(string(match[1]))
	if err != nil {
		return nil
	}
	return &units
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
