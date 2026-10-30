package tcgmatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

func (c Client) Search(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	name := query.Name
	base, err := url.Parse(c.BaseURL)
	if err != nil || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, errors.New("invalid TCGMatch URL")
	}
	if strings.TrimSpace(name) == "" || c.Game == "" {
		return nil, errors.New("invalid TCGMatch search")
	}
	products, err := c.searchCatalog(ctx, base, name, catalogType(query.Kind))
	if err != nil {
		return nil, err
	}
	if c.Game == "pokemon" && !query.Sealed() {
		c.namePokemonFunctions(ctx, products)
	}
	items := make([]model.Offer, 0)
	for _, product := range products {
		if product.TCG != c.Game || product.Type != catalogType(query.Kind) || !query.AcceptsTitle(product.Name) {
			continue
		}
		listings, err := c.readListings(ctx, base, product.ID)
		if err != nil {
			return nil, err
		}
		for _, entry := range listings {
			if item, ok := buildOffer(entry, product, c.Game, query.Kind); ok {
				items = append(items, item)
			}
		}
	}
	return model.DeduplicateOffers(items), nil
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

func buildOffer(entry listing, product catalogProduct, game string, kind model.ProductKind) (model.Offer, bool) {
	if !entry.IsActive || entry.Quantity < 1 || entry.Price < 1 || entry.ID == "" || entry.User.Name == "" || entry.TCG != game {
		return model.Offer{}, false
	}
	finish := "nonfoil"
	if entry.IsHoloReverse {
		finish = "reverse-holo"
	} else if entry.IsHolo {
		finish = "holo"
	}
	return model.Offer{
		ID: "tcgmatch:" + entry.ID, VariantID: entry.ID, CardName: product.Name,
		Store: entry.User.Name, PriceAmount: strconv.FormatInt(entry.Price, 10), PriceCurrency: "CLP",
		URL: "https://tcgmatch.cl/producto/" + entry.ID, Image: product.Image, Source: "tcgmatch.cl", StockStatus: "available",
		Language: entry.Language, Condition: entry.Status, Finish: finish, Edition: product.SetCode, Kind: kind,
		Metadata: map[string]string{
			"game": game, "quantity": strconv.Itoa(entry.Quantity), "seller": entry.User.Username,
			"product_id": strconv.FormatInt(product.ID, 10), "set_id": product.SetID,
			"set_code": product.SetCode, "card_code": product.CardCode, "image": product.Image,
			"functional_key": product.Function,
		},
	}, true
}
