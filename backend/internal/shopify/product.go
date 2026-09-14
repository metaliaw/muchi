package shopify

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type variantReply struct {
	ID        int64    `json:"id"`
	Title     string   `json:"title"`
	Price     *int64   `json:"price"`
	Available bool     `json:"available"`
	Options   []string `json:"options"`
	SKU       string   `json:"sku"`
}

type productReply struct {
	ID       int64          `json:"id"`
	Title    string         `json:"title"`
	Variants []variantReply `json:"variants"`
	Options  []struct {
		Name     string `json:"name"`
		Position int    `json:"position"`
	} `json:"options"`
}

func (c Client) buildOffers(product productReply, path, name, currency string) ([]offer.Offer, error) {
	items := make([]offer.Offer, 0, len(product.Variants))
	for _, variant := range product.Variants {
		if !variant.Available {
			continue
		}
		if variant.ID <= 0 || variant.Price == nil || *variant.Price <= 0 {
			return nil, errors.New("invalid Shopify variant price or ID")
		}
		id := strconv.FormatInt(variant.ID, 10)
		store := c.Name
		if store == "" {
			store = c.Domain
		}
		item := offer.Offer{
			ID: c.Domain + ":" + id, VariantID: id, CardName: name, Store: store,
			PriceAmount: formatPrice(*variant.Price), PriceCurrency: currency,
			URL: "https://" + c.Domain + path + "?variant=" + id, Source: c.Domain, StockStatus: "available",
			Metadata: map[string]string{"title": product.Title, "variant": variant.Title, "sku": variant.SKU, "product_id": strconv.FormatInt(product.ID, 10)},
		}
		applyOptions(&item, product, variant)
		items = append(items, item)
	}
	return items, nil
}

// formatPrice Removes Shopify's Hundredths, including for Zero-Decimal Currencies.
func formatPrice(price int64) string {
	if price%100 == 0 {
		return strconv.FormatInt(price/100, 10)
	}
	return fmt.Sprintf("%d.%02d", price/100, price%100)
}

func applyOptions(item *offer.Offer, product productReply, variant variantReply) {
	for _, option := range product.Options {
		index := option.Position - 1
		if index < 0 || index >= len(variant.Options) {
			continue
		}
		value := variant.Options[index]
		switch strings.ToLower(option.Name) {
		case "language", "idioma":
			item.Language = value
		case "condition", "condición", "condicion", "estado":
			item.Condition = value
		case "finish", "acabado", "foil":
			item.Finish = value
		}
	}
}

func stockStatus(available bool) string {
	if available {
		return "available"
	}
	return "unavailable"
}
