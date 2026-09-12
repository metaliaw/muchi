package tcgmatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

const catalogLimit = 24
const autocompleteLimit = 3

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

type Client struct {
	Fetcher SourceFetcher
	BaseURL string
	Game    string
}

type catalogReply struct {
	Products []catalogProduct `json:"products"`
}

type catalogProduct struct {
	ID        int64    `json:"id"`
	Name      string   `json:"name"`
	TCG       string   `json:"tcg"`
	Type      string   `json:"type"`
	Image     string   `json:"image"`
	SetID     string   `json:"setId"`
	SetCode   string   `json:"setCode"`
	Languages []string `json:"languages"`
}

type listingsReply struct {
	Success bool      `json:"success"`
	Data    []listing `json:"data"`
}

type listing struct {
	ID            string `json:"_id"`
	Name          string `json:"name"`
	TCG           string `json:"tcg"`
	Language      string `json:"language"`
	Status        string `json:"status"`
	Quantity      int    `json:"quantity"`
	Price         int64  `json:"price"`
	IsActive      bool   `json:"isActive"`
	IsHolo        bool   `json:"isHolo"`
	IsHoloReverse bool   `json:"isHoloReverse"`
	User          struct {
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"user"`
}

func (c Client) Search(ctx context.Context, name string) ([]offer.Offer, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid TCGMatch URL")
	}
	if strings.TrimSpace(name) == "" || c.Game == "" {
		return nil, errors.New("invalid TCGMatch search")
	}
	products, err := c.searchCatalog(ctx, base, name)
	if err != nil {
		return nil, err
	}
	items := make([]offer.Offer, 0)
	for _, product := range products {
		if product.TCG != c.Game || product.Type != "card" {
			continue
		}
		listings, err := c.readListings(ctx, base, product.ID)
		if err != nil {
			return nil, err
		}
		for _, entry := range listings {
			if item, ok := buildOffer(entry, product.Name, c.Game); ok {
				items = append(items, item)
			}
		}
	}
	return offer.DeduplicateOffers(items), nil
}

func (c Client) CardMetadata(ctx context.Context, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	base, err := c.validBase(request.Name)
	if err != nil {
		return cardmetadata.Metadata{}, err
	}
	products, err := c.searchCatalog(ctx, base, request.Name)
	if err != nil {
		return cardmetadata.Metadata{}, err
	}
	for _, product := range products {
		if !c.matches(product, request.Language) || request.Edition != "" && !strings.EqualFold(request.Edition, product.SetCode) && !strings.EqualFold(request.Edition, product.SetID) {
			continue
		}
		return cardmetadata.Metadata{Name: product.Name, Image: product.Image}, nil
	}
	return cardmetadata.Metadata{}, nil
}

func (c Client) Autocomplete(ctx context.Context, text, language string) ([]string, error) {
	base, err := c.validBase(text)
	if err != nil {
		return nil, err
	}
	products, err := c.searchCatalog(ctx, base, text)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, autocompleteLimit)
	seen := map[string]bool{}
	for _, product := range products {
		if !c.matches(product, language) || seen[product.Name] {
			continue
		}
		names = append(names, product.Name)
		seen[product.Name] = true
		if len(names) == autocompleteLimit {
			break
		}
	}
	return names, nil
}

func (c Client) validBase(text string) (*url.URL, error) {
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid TCGMatch URL")
	}
	if strings.TrimSpace(text) == "" || c.Game == "" {
		return nil, errors.New("invalid TCGMatch search")
	}
	return base, nil
}

func (c Client) matches(product catalogProduct, language string) bool {
	if product.TCG != c.Game || product.Type != "card" {
		return false
	}
	if language == "" {
		return true
	}
	for _, available := range product.Languages {
		if strings.EqualFold(available, language) {
			return true
		}
	}
	return false
}

func (c Client) searchCatalog(ctx context.Context, base *url.URL, name string) ([]catalogProduct, error) {
	query := url.Values{"q": {name}, "tcg": {c.Game}, "type": {"card"}, "inStock": {"true"}, "limit": {strconv.Itoa(catalogLimit)}, "page": {"1"}, "sortBy": {"listings"}}
	target := strings.TrimRight(c.BaseURL, "/") + "/catalog/search?" + query.Encode()
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		return nil, err
	}
	var reply catalogReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, fmt.Errorf("decode TCGMatch catalog: %w", err)
	}
	return reply.Products, nil
}

func (c Client) readListings(ctx context.Context, base *url.URL, catalogID int64) ([]listing, error) {
	target := fmt.Sprintf("%s/products/catalog/%d?inStock=true", strings.TrimRight(c.BaseURL, "/"), catalogID)
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		return nil, err
	}
	var reply listingsReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, fmt.Errorf("decode TCGMatch listings: %w", err)
	}
	if !reply.Success {
		return nil, errors.New("invalid TCGMatch listings")
	}
	return reply.Data, nil
}

func buildOffer(entry listing, cardName, game string) (offer.Offer, bool) {
	if !entry.IsActive || entry.Quantity < 1 || entry.Price < 1 || entry.ID == "" || entry.User.Name == "" || entry.TCG != game {
		return offer.Offer{}, false
	}
	finish := "nonfoil"
	if entry.IsHoloReverse {
		finish = "reverse-holo"
	} else if entry.IsHolo {
		finish = "holo"
	}
	return offer.Offer{
		ID: "tcgmatch:" + entry.ID, VariantID: entry.ID, CardName: cardName,
		Store: entry.User.Name, PriceAmount: strconv.FormatInt(entry.Price, 10), PriceCurrency: "CLP",
		URL: "https://tcgmatch.cl/producto/" + entry.ID, Source: "tcgmatch.cl", StockStatus: "available",
		Language: entry.Language, Condition: entry.Status, Finish: finish,
		Metadata: map[string]string{"game": game, "quantity": strconv.Itoa(entry.Quantity), "seller": entry.User.Username},
	}, true
}
