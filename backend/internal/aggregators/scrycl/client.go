package scrycl

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

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher SourceFetcher
	BaseURL string
	// ExcludeCommunity Drops the Offers Scry.cl hosts itself: a Community Seller
	// holds a page under the Scry.cl Domain, never a Storefront of its own.
	ExcludeCommunity bool
}

func (c Client) Search(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	return c.FindOffers(ctx, query)
}

var slugSeparators = regexp.MustCompile(`[^a-z0-9]+`)

// FindOffers Reads the Saved Offers Published on a Scry.cl Card Page.
func (c Client) FindOffers(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	name := query.Name
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid Scry.cl URL")
	}
	slug := buildCardSlug(name)
	if slug == "" {
		return nil, errors.New("empty Scry.cl card slug")
	}
	target := strings.TrimRight(c.BaseURL, "/") + "/card/" + slug
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		var status source.StatusError
		if errors.As(err, &status) && status.Code == 404 {
			return []model.Offer{}, nil
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

func readOffers(data []byte) ([]model.Offer, error) {
	document, err := html.Parse(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode Scry.cl page: %w", err)
	}
	items := make([]model.Offer, 0)
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
		return nil, errors.New("Scry.cl page missing results container")
	}
	return model.DeduplicateOffers(items), nil
}

func readAttributes(node *html.Node) map[string]string {
	attrs := make(map[string]string, len(node.Attr))
	for _, attr := range node.Attr {
		attrs[attr.Key] = attr.Val
	}
	return attrs
}

// withoutCommunityOffers Keeps the Offers that link out to a Storefront. A
// Community Offer points back under the Scry.cl Domain —marketplace.scry.cl—
// because the Seller has no Site of its own to link to.
func withoutCommunityOffers(items []model.Offer, host string) []model.Offer {
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

func buildOffer(attrs map[string]string) (model.Offer, error) {
	item := model.Offer{
		CardName: attrs["data-card-name"], Store: attrs["data-store-name"],
		PriceAmount: attrs["data-price-clp"], PriceCurrency: "CLP",
		URL: attrs["data-product-url"], VariantID: attrs["data-variant-key"],
		Source: "scry.cl", StockStatus: "unknown",
		Metadata: map[string]string{"title": attrs["data-offer-title"]},
	}
	price, err := strconv.ParseInt(item.PriceAmount, 10, 64)
	if err != nil || price <= 0 || item.CardName == "" || model.ValidateOffer(item) != nil {
		return model.Offer{}, errors.New("invalid Scry.cl offer")
	}
	identity := strings.Join([]string{item.Store, item.URL, item.VariantID}, "|")
	item.ID = fmt.Sprintf("scry:%x", sha256.Sum256([]byte(identity)))
	return item, nil
}

// SourceName Identifies this Provider the Way the Health Report Names it.
func (c Client) SourceName() string {
	if base, err := url.Parse(c.BaseURL); err == nil && base.Host != "" {
		return base.Host
	}
	return c.BaseURL
}
