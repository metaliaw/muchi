package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

// SourceFetcher Reads Store pages interpreted with https://schema.org/availability.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// StockReader Checks Stock according to one Store Platform.
type StockReader interface {
	CheckStock(context.Context, model.Offer) (model.StockReading, error)
}

// CheckStock Asks the Store's own Platform when it can Answer, and Reads the
// Product Page against the Store's Configuration when it cannot.
func (s Shops) CheckStock(ctx context.Context, item model.Offer) (model.StockReading, error) {
	if s.Lists != nil && item.Source == "moxfield" {
		return s.Lists.CheckStock(ctx, item)
	}
	link, err := url.Parse(item.URL)
	if err != nil {
		return model.ReadStock("unknown"), err
	}
	shop, found := s.Shop(link.Host)
	if !found {
		return model.ReadStock("unknown"), nil
	}
	if adapter, can := shop.Client.(StockReader); can {
		return adapter.CheckStock(ctx, item)
	}
	data, err := s.Fetcher.FetchSource(ctx, link.Host, item.URL)
	if err != nil {
		return model.ReadStock("unknown"), err
	}
	status := inspectStock(data, shop.Config)
	units := model.CountDeclaredUnits(data)
	if units == nil {
		return model.ReadStock(status), nil
	}
	// A Page that Counts its Units has Answered the Question: none Left is
	// Sold out, and any Number is a Store Saying yes with a Figure behind it.
	if *units == 0 {
		return model.CountStock("unavailable", 0), nil
	}
	if status == "unavailable" {
		return model.CountStock("unavailable", 0), nil
	}
	return model.CountStock("available", *units), nil
}

func inspectStock(data []byte, config directory.StoreConfig) string {
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
