package jumpseller

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type stockReply struct {
	ID        int64  `json:"id"`
	Stock     *int   `json:"stock"`
	Unlimited bool   `json:"stock_unlimited"`
	Status    string `json:"status"`
}
type formReply struct {
	Info struct {
		Product struct {
			stockReply
			Name    string `json:"name"`
			URL     string `json:"url"`
			Options []struct {
				ID   int64  `json:"id"`
				Name string `json:"name"`
			} `json:"options"`
		} `json:"product"`
		Variant struct {
			Price json.Number `json:"price_with_discount"`
		} `json:"variant"`
	} `json:"info"`
}
type variantReply struct {
	Variant struct {
		stockReply
		SKU       string `json:"sku"`
		ProductID int64  `json:"product_id"`
	} `json:"variant"`
	Price    json.Number `json:"price"`
	Discount json.Number `json:"discount"`
	Status   string      `json:"status"`
	Values   []struct {
		Value struct {
			Name   string `json:"name"`
			Option int64  `json:"option"`
		} `json:"value"`
	} `json:"values"`
}

func (c Client) parseProduct(data []byte, path string) ([]offer.Offer, error) {
	form, currency, variants, err := readProduct(data, path, c.Domain)
	if err != nil {
		return nil, err
	}
	base := offer.Offer{CardName: form.Info.Product.Name, Store: c.Name, Source: c.Domain, PriceCurrency: currency, URL: "https://" + c.Domain + path, Metadata: map[string]string{"title": form.Info.Product.Name}}
	if base.Store == "" {
		base.Store = c.Domain
	}
	if len(variants) == 0 {
		if len(form.Info.Product.Options) > 0 {
			return nil, errors.New("Jumpseller product missing variants")
		}
		base.ID = c.Domain + ":" + strconv.FormatInt(form.Info.Product.ID, 10)
		base.PriceAmount, err = priceAmount(form.Info.Variant.Price, "0")
		if err != nil {
			return nil, err
		}
		base.StockStatus = stockStatus(form.Info.Product.stockReply)
		base.StockQuantity = countStock(form.Info.Product.stockReply)
		return []offer.Offer{base}, nil
	}
	return buildVariants(base, form, variants)
}

func buildVariants(base offer.Offer, form formReply, variants []variantReply) ([]offer.Offer, error) {
	items := make([]offer.Offer, 0, len(variants))
	for _, variant := range variants {
		if variant.Variant.ID <= 0 || variant.Variant.ProductID != form.Info.Product.ID {
			return nil, errors.New("invalid Jumpseller variant ID")
		}
		item := base
		item.VariantID = strconv.FormatInt(variant.Variant.ID, 10)
		item.ID = base.Source + ":" + item.VariantID
		item.URL = base.URL + "?variant_id=" + item.VariantID
		amount, err := priceAmount(variant.Price, variant.Discount)
		if err != nil {
			return nil, err
		}
		item.PriceAmount = amount
		stock := variant.Variant.stockReply
		stock.Status = variant.Status
		if form.Info.Product.Status != "available" {
			stock.Status = form.Info.Product.Status
		}
		item.StockStatus = stockStatus(stock)
		item.StockQuantity = countStock(stock)
		item.Metadata = map[string]string{"title": base.CardName, "sku": variant.Variant.SKU}
		applyOptions(&item, form, variant)
		items = append(items, item)
	}
	return items, nil
}

func applyOptions(item *offer.Offer, form formReply, variant variantReply) {
	for _, option := range form.Info.Product.Options {
		for _, entry := range variant.Values {
			if entry.Value.Option != option.ID {
				continue
			}
			value := entry.Value.Name
			switch strings.ToLower(option.Name) {
			case "idioma", "lenguaje", "language":
				item.Language = value
			case "condición", "condicion", "condition", "estado":
				item.Condition = value
			case "acabado", "finish", "foil":
				item.Finish = value
			}
		}
	}
}

// countStock Keeps the Number only when the Store Kept one. An Unlimited
// Stock Counts nothing: the Store Sells without Counting, and inventing a
// Number there would Say more than the Store Said.
func countStock(stock stockReply) *int {
	if stock.Unlimited || stock.Stock == nil {
		return nil
	}
	units := *stock.Stock
	return &units
}

func stockStatus(stock stockReply) string {
	if stock.Status == "disabled" || stock.Status == "not-available" || stock.Status == "out-of-stock" {
		return "unavailable"
	}
	if stock.Status != "available" {
		return "unknown"
	}
	if stock.Unlimited {
		return "available"
	}
	if stock.Stock == nil {
		return "unknown"
	}
	if *stock.Stock > 0 {
		return "available"
	}
	return "unavailable"
}

// priceAmount Preserves Decimal Units and Subtracts the Published Discount.
func priceAmount(price, discount json.Number) (string, error) {
	amount, ok := new(big.Rat).SetString(string(price))
	if !ok {
		return "", errors.New("invalid Jumpseller price")
	}
	reduction, ok := new(big.Rat).SetString(string(discount))
	if !ok || reduction.Sign() < 0 {
		return "", errors.New("invalid Jumpseller discount")
	}
	amount.Sub(amount, reduction)
	if amount.Sign() <= 0 {
		return "", errors.New("invalid Jumpseller net price")
	}
	if amount.IsInt() {
		return amount.Num().String(), nil
	}
	return amount.FloatString(2), nil
}
