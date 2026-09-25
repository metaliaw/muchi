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
	// Sessions Lets QuoteCart Fill a Cart; Searching Never Needs it.
	Sessions SessionSender
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
func (c Client) CheckStock(ctx context.Context, item offer.Offer) (offer.StockReading, error) {
	link, err := url.Parse(item.URL)
	if err != nil {
		return offer.ReadStock("unknown"), err
	}
	path := productPath(link, c.Domain)
	if path == "" {
		return offer.ReadStock("unknown"), errors.New("invalid Shopify product URL")
	}
	variant := link.Query().Get("variant")
	if variant == "" {
		variant = item.VariantID
	}
	if variant == "" {
		return offer.ReadStock("unknown"), nil
	}
	product, err := c.readProduct(ctx, path)
	if err != nil {
		return offer.ReadStock("unknown"), err
	}
	for _, candidate := range product.Variants {
		if strconv.FormatInt(candidate.ID, 10) != variant {
			continue
		}
		if !candidate.Available {
			return offer.CountStock("unavailable", 0), nil
		}
		// The Ajax Product Answers yes or no; the Page Counts. A Buyer Adding
		// Copies Needs the Number, so the Page is Read once the Answer is yes.
		if units := c.countUnits(ctx, path, variant); units != nil && *units > 0 {
			return offer.CountStock("available", *units), nil
		}
		return offer.ReadStock("available"), nil
	}
	return offer.ReadStock("unknown"), nil
}

// countUnits Reads the Storefront Page for the Count the Theme Prints. A Page
// that will not Load Costs the Number, never the Answer: the Ajax Product
// already Said the Variant Sells.
func (c Client) countUnits(ctx context.Context, path, variant string) *int {
	// The Page is Asked for this Variant: a Theme Renders its Counter for the
	// Variant Selected, and a Page without one Answers about the wrong Card.
	data, err := c.fetchPage(ctx, path+"?variant="+variant)
	if err != nil {
		return nil
	}
	return readVariantUnits(data, variant)
}

// SourceName Identifies this Store the Way the Health Report Names it.
func (c Client) SourceName() string { return c.Domain }
