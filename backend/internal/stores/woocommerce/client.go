package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type SourceFetcher interface {
	FetchSource(context.Context, string, string) ([]byte, error)
}

// Client Reads the Public WooCommerce Store API for a Configured Store.
type Client struct {
	Fetcher SourceFetcher
	Domain  string
	Name    string
	// Sessions Lets QuoteCart Fill a Cart; Searching Never Needs it.
	Sessions SessionSender
}

type productReply struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	URL     string `json:"permalink"`
	InStock bool   `json:"is_in_stock"`
	Prices  struct {
		Price     string `json:"price"`
		Currency  string `json:"currency_code"`
		MinorUnit int    `json:"currency_minor_unit"`
	} `json:"prices"`
}

func (c Client) FindOffers(ctx context.Context, query model.CardQuery) ([]model.Offer, error) {
	name := query.Name
	items := make([]model.Offer, 0)
	for page := 1; ; page++ {
		target := fmt.Sprintf("https://%s/wp-json/wc/store/v1/products?search=%s&per_page=100&page=%d", c.Domain, url.QueryEscape(name), page)
		data, err := c.Fetcher.FetchSource(ctx, c.Domain, target)
		if err != nil {
			return nil, err
		}
		var products []productReply
		if err := json.Unmarshal(data, &products); err != nil {
			return nil, fmt.Errorf("decode store catalog %s: %w", c.Domain, err)
		}
		for _, product := range products {
			if !query.AcceptsTitle(html.UnescapeString(product.Name)) {
				continue
			}
			item, err := c.buildOffer(product)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		if len(products) < 100 {
			return items, nil
		}
	}
}

func (c Client) buildOffer(product productReply) (model.Offer, error) {
	price, err := formatPrice(product.Prices.Price, product.Prices.MinorUnit)
	if err != nil {
		return model.Offer{}, err
	}
	store := c.Name
	if store == "" {
		store = c.Domain
	}
	status := "unavailable"
	if product.InStock {
		status = "available"
	}
	item := model.Offer{
		ID: fmt.Sprintf("%s:%d", c.Domain, product.ID), CardName: html.UnescapeString(product.Name),
		Store: store, PriceAmount: price, PriceCurrency: product.Prices.Currency,
		URL: product.URL, Source: c.Domain, StockStatus: status,
	}
	// Only a Simple Product Goes in a Cart by its own Id; a Variable one
	// Needs the Variation.
	if product.Type == "simple" {
		item.VariantID = strconv.Itoa(product.ID)
	}
	if item.PriceCurrency == "" || model.ValidateOffer(item) != nil {
		return model.Offer{}, fmt.Errorf("invalid store offer from %s", c.Domain)
	}
	return item, nil
}

func formatPrice(value string, minor int) (string, error) {
	amount, err := strconv.ParseUint(value, 10, 64)
	if err != nil || amount == 0 || minor < 0 || minor > 6 {
		return "", fmt.Errorf("invalid store price %q", value)
	}
	if minor == 0 {
		return value, nil
	}
	digits := strings.Repeat("0", max(0, minor+1-len(value))) + value
	point := len(digits) - minor
	return digits[:point] + "." + digits[point:], nil
}

// SourceName Identifies this Store the Way the Health Report Names it.
func (c Client) SourceName() string { return c.Domain }
