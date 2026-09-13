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

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher SourceFetcher
	BaseURL string
	Prints  cardmetadata.PrintProvider
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
	items = c.applyPrintImages(ctx, name, items)
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

func (c Client) applyPrintImages(ctx context.Context, name string, items []offer.Offer) []offer.Offer {
	if c.Prints == nil || len(items) == 0 {
		return items
	}
	prints, err := c.Prints.CardPrints(ctx, name)
	if err != nil {
		return items
	}
	exact, editions := indexPrintImages(prints)
	for index := range items {
		items[index].Image = matchPrintImage(exact, editions, items[index].Metadata["title"], name)
	}
	return items
}

func indexPrintImages(prints []cardmetadata.Print) (map[string]string, map[string]string) {
	exact := make(map[string]string, len(prints))
	editions := make(map[string]string)
	for _, print := range prints {
		number := strings.ToLower(print.CollectorNumber)
		if number == "" || print.Image == "" {
			continue
		}
		// Una Tienda Escribe `[C21]` y otra `[Fallout]`: el Código y el
		// Nombre Nombran la misma Edición, y Scryfall Trae los dos.
		// Indexar ambos Deja que el Título Elija el Idioma que Prefiera.
		for _, edition := range []string{strings.ToLower(print.Edition), strings.ToLower(print.EditionName)} {
			if edition == "" {
				continue
			}
			exact[edition+":"+number] = print.Image
			// La primera Impresión que Scryfall Lista Representa a su
			// Edición cuando la Oferta Calla el Número. Una Hermana
			// Muestra la Carta; un Hueco no Muestra nada.
			if _, seen := editions[edition]; !seen {
				editions[edition] = print.Image
			}
		}
	}
	return exact, editions
}

// matchPrintImage Finds the Image for whatever Shape a Store Chose to Write.
// Every Store Writes its own: `[C21] #263`, `| C20`, `(C19 #221)`,
// `[C14-270]`, `[MKC - 237]`, `- 214 -`. Guessing the Shape Needs one Rule
// per Store; Asking instead which Word *is* an Edition Scryfall Listed for
// this Card Needs none, and Never Invents one.
func matchPrintImage(exact, editions map[string]string, title, name string) string {
	words := titleWords(strings.TrimPrefix(strings.ToLower(title), strings.ToLower(name)))
	edition := longestEdition(words, editions)
	if edition == "" {
		return ""
	}
	// Every numeric Word is a Candidate Number, and a wrong Guess Costs
	// nothing: only a Pair the Print List Confirms Wins. When none does,
	// the Edition alone Answers with its first Impresión.
	for _, word := range words {
		if image := exact[edition+":"+word]; image != "" {
			return image
		}
	}
	return editions[edition]
}

var wordPattern = regexp.MustCompile(`[\p{L}\p{N}★]+`)

func titleWords(title string) []string {
	return wordPattern.FindAllString(title, -1)
}

// longestEdition Prefers `modern horizons 2` over a bare `2`, so a Name of
// several Words Wins against a Fragment of itself.
func longestEdition(words []string, editions map[string]string) string {
	const longestEditionName = 6
	for length := longestEditionName; length >= 1; length-- {
		for start := 0; start+length <= len(words); start++ {
			candidate := strings.Join(words[start:start+length], " ")
			if _, known := editions[candidate]; known {
				return candidate
			}
		}
	}
	return ""
}
