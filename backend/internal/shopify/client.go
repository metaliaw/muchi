package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// SourceFetcher Reads Storefront Pages and https://shopify.dev/docs/api/ajax/reference/product Responses.
type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// Client Reads Public Storefront Search and Ajax Product Responses.
type Client struct {
	Fetcher SourceFetcher
	Domain  string
	Name    string
}

func (c Client) FindOffers(ctx context.Context, query offer.CardQuery) ([]offer.Offer, error) {
	name := query.Name
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("empty card name")
	}
	links, err := c.findProducts(ctx, name)
	if err != nil {
		return nil, err
	}
	items := make([]offer.Offer, 0)
	if len(links) == 0 {
		return items, nil
	}
	currency, err := c.readCurrency(ctx)
	if err != nil {
		return nil, err
	}
	for _, link := range links {
		product, err := c.readProduct(ctx, link)
		if err != nil {
			return nil, err
		}
		if !query.AcceptsTitle(product.Title) {
			continue
		}
		offers, err := c.buildOffers(product, link, currency)
		if err != nil {
			return nil, err
		}
		items = append(items, offers...)
	}
	return offer.DeduplicateOffers(items), nil
}

func (c Client) fetchPage(ctx context.Context, path string) ([]byte, error) {
	return c.Fetcher.FetchSource(ctx, c.Domain, "https://"+c.Domain+path)
}

func (c Client) readCurrency(ctx context.Context) (string, error) {
	data, err := c.fetchPage(ctx, "/cart.js")
	if err != nil {
		return "", err
	}
	var cart struct {
		Currency string `json:"currency"`
	}
	if err := json.Unmarshal(data, &cart); err != nil {
		return "", fmt.Errorf("decode Shopify currency: %w", err)
	}
	if len(cart.Currency) != 3 || strings.Trim(cart.Currency, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") != "" {
		return "", errors.New("invalid Shopify currency")
	}
	return cart.Currency, nil
}

func (c Client) readProduct(ctx context.Context, path string) (productReply, error) {
	var product productReply
	data, err := c.fetchPage(ctx, path+".js")
	if err != nil {
		return product, err
	}
	if err := json.Unmarshal(data, &product); err != nil {
		return product, fmt.Errorf("decode Shopify product: %w", err)
	}
	if product.ID <= 0 || product.Title == "" || product.Variants == nil {
		return product, errors.New("invalid Shopify product")
	}
	return product, nil
}

// CheckStock Reads the Requested Variant, including Variants that Sold Out.
func (c Client) CheckStock(ctx context.Context, item offer.Offer) (string, error) {
	link, err := url.Parse(item.URL)
	if err != nil {
		return "unknown", err
	}
	path := productPath(link, c.Domain)
	if path == "" {
		return "unknown", errors.New("invalid Shopify product URL")
	}
	variant := link.Query().Get("variant")
	if variant == "" {
		variant = item.VariantID
	}
	if variant == "" {
		return "unknown", nil
	}
	product, err := c.readProduct(ctx, path)
	if err != nil {
		return "unknown", err
	}
	for _, candidate := range product.Variants {
		if strconv.FormatInt(candidate.ID, 10) == variant {
			return stockStatus(candidate.Available), nil
		}
	}
	return "unknown", nil
}

// SourceName Identifies this Store the Way the Health Report Names it.
func (c Client) SourceName() string { return c.Domain }
