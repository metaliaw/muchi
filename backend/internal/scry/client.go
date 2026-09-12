package scry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/text/unicode/norm"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher SourceFetcher
	BaseURL string
	// ExcludeCommunity Drops the Offers Scry hosts itself: a Community Seller
	// holds a page under the Scry Domain, never a Storefront of its own.
	ExcludeCommunity bool
}

func (c Client) Search(ctx context.Context, name string) ([]offer.Offer, error) {
	return c.FindOffers(ctx, name)
}

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

// FindOffers Reads the Saved Offers Published on a Scry Card Page.
func (c Client) FindOffers(ctx context.Context, name string) ([]offer.Offer, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid Scry URL")
	}
	slug := buildCardSlug(name)
	if slug == "" {
		return nil, errors.New("empty Scry card slug")
	}
	target := strings.TrimRight(c.BaseURL, "/") + "/card/" + slug
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		var status source.StatusError
		if errors.As(err, &status) && status.Code == 404 {
			return []offer.Offer{}, nil
		}
		return nil, err
	}
	items, err := readOffers(data)
	if err != nil {
		return nil, err
	}
	if c.ExcludeCommunity {
		items = withoutCommunityOffers(items, base.Host)
	}
	return items, nil
}

func buildCardSlug(name string) string {
	decomposed := norm.NFKD.String(name)
	plain := strings.Map(func(r rune) rune {
		if r >= '\u0300' && r <= '\u036f' {
			return -1
		}
		return r
	}, decomposed)
	return strings.Trim(slugSeparators.ReplaceAllString(strings.ToLower(plain), "-"), "-")
}

func readOffers(data []byte) ([]offer.Offer, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode Scry page: %w", err)
	}
	items := make([]offer.Offer, 0)
	found := false
	for node := range document.Descendants() {
		attrs := readAttributes(node)
		if attrs["id"] == "results" {
			found = true
		}
		if node.Data != "a" || attrs["data-track-type"] != "store_offer_click" {
			continue
		}
		item, err := buildOffer(attrs)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if !found {
		return nil, errors.New("Scry page missing results container")
	}
	return offer.DeduplicateOffers(items), nil
}

func readAttributes(node *html.Node) map[string]string {
	attrs := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

// withoutCommunityOffers Keeps the Offers that link out to a Storefront. A
// Community Offer points back under the Scry Domain —marketplace.scry.cl—
// because the Seller has no Site of its own to link to.
func withoutCommunityOffers(items []offer.Offer, host string) []offer.Offer {
	kept := items[:0]
	for _, item := range items {
		link, err := url.Parse(item.URL)
		if err != nil || link.Host == host || strings.HasSuffix(link.Host, "."+host) {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}

func buildOffer(attrs map[string]string) (offer.Offer, error) {
	item := offer.Offer{
		CardName: attrs["data-card-name"], Store: attrs["data-store-name"],
		PriceAmount: attrs["data-price-clp"], PriceCurrency: "CLP",
		URL: attrs["data-product-url"], VariantID: attrs["data-variant-key"],
		Source: "scry.cl", StockStatus: "unknown",
		Metadata: map[string]string{"title": attrs["data-offer-title"]},
	}
	price, err := strconv.ParseInt(item.PriceAmount, 10, 64)
	if err != nil || price <= 0 || item.CardName == "" || offer.ValidateOffer(item) != nil {
		return offer.Offer{}, errors.New("invalid Scry offer")
	}
	identity := strings.Join([]string{item.Store, item.URL, item.VariantID}, "|")
	item.ID = fmt.Sprintf("scry:%x", sha256.Sum256([]byte(identity)))
	return item, nil
}
