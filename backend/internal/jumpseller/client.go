package jumpseller

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// SourceFetcher Reads Storefront Pages described by https://jumpseller.com/support/liquid/.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}
type Client struct {
	Fetcher      SourceFetcher
	Domain, Name string
}

func (c Client) FindOffers(ctx context.Context, query offer.CardQuery) ([]offer.Offer, error) {
	name := query.Name
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("empty card name")
	}
	paths, err := c.findProducts(ctx, name)
	if err != nil {
		return nil, err
	}
	items := make([]offer.Offer, 0)
	for _, path := range paths {
		products, err := c.readOffers(ctx, path)
		if err != nil {
			return nil, err
		}
		for _, item := range products {
			if offer.MatchesCard(item.CardName, name) && item.StockStatus == "available" {
				items = append(items, item)
			}
		}
	}
	return offer.DeduplicateOffers(items), nil
}

func (c Client) readOffers(ctx context.Context, path string) ([]offer.Offer, error) {
	data, err := c.fetchPage(ctx, path)
	if err != nil {
		return nil, err
	}
	return c.parseProduct(data, path)
}

func (c Client) CheckStock(ctx context.Context, item offer.Offer) (string, error) {
	link, err := url.Parse(item.URL)
	if err != nil {
		return "unknown", err
	}
	if !sameStore(link, c.Domain) || link.Path == "" {
		return "unknown", errors.New("invalid Jumpseller product URL")
	}
	items, err := c.readOffers(ctx, link.EscapedPath())
	if err != nil {
		return "unknown", err
	}
	variant := link.Query().Get("variant_id")
	if variant == "" {
		variant = item.VariantID
	}
	for _, candidate := range items {
		if candidate.VariantID == variant || (variant == "" && len(items) == 1) {
			return candidate.StockStatus, nil
		}
	}
	return "unknown", nil
}

// requestGate Serializes Jumpseller Reads within one Process.
var requestGate = make(chan struct{}, 1)

// fetchPage Spaces Storefront Reads because Search and Products Share Rate Limits.
func (c Client) fetchPage(ctx context.Context, path string) ([]byte, error) {
	select {
	case requestGate <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { <-requestGate }()
	timer := time.NewTimer(4 * time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
	}
	if strings.HasPrefix(path, "/api/") {
		if fetcher, ok := c.Fetcher.(interface {
			FetchStorefront(context.Context, string, string) ([]byte, error)
		}); ok {
			return fetcher.FetchStorefront(ctx, c.Domain, "https://"+c.Domain+path)
		}
	}
	return c.Fetcher.FetchSource(ctx, c.Domain, "https://"+c.Domain+path)
}
