package woocommerce

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strconv"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// SessionSender Makes the Calls a Cart Lives in.
type SessionSender interface {
	SendSession(context.Context, string, source.SessionRequest) ([]byte, http.Header, error)
}

// ErrNotQuotable Answers a Line whose Product the Cart cannot Name.
var ErrNotQuotable = offer.ErrNoQuote

type cartReply struct {
	Items []struct {
		ID       int `json:"id"`
		Quantity int `json:"quantity"`
		Prices   struct {
			Price     string `json:"price"`
			MinorUnit int    `json:"currency_minor_unit"`
		} `json:"prices"`
	} `json:"items"`
	Totals struct {
		Items     string `json:"total_items"`
		Shipping  string `json:"total_shipping"`
		Total     string `json:"total_price"`
		Currency  string `json:"currency_code"`
		MinorUnit int    `json:"currency_minor_unit"`
	} `json:"totals"`
	Packages []struct {
		Rates []struct {
			ID       string `json:"rate_id"`
			Name     string `json:"name"`
			Price    string `json:"price"`
			Selected bool   `json:"selected"`
		} `json:"shipping_rates"`
	} `json:"shipping_rates"`
	PaymentMethods []string `json:"payment_methods"`
	Errors         []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// QuoteCart Fills a Fresh Store API Cart with the Lines, Gives it the Address,
// and Reads back what the Store would Charge. It Stops before Checkout: no
// Order is Created and no Stock is Held. The Cart is Left to Expire.
//
// https://github.com/woocommerce/woocommerce/blob/trunk/plugins/woocommerce/src/StoreApi/docs/cart.md
func (c Client) QuoteCart(ctx context.Context, lines []offer.CartLine, address offer.ShippingAddress) (offer.CartQuote, error) {
	if c.Sessions == nil || len(lines) == 0 {
		return offer.CartQuote{}, ErrNotQuotable
	}
	products := make([]int, len(lines))
	for index, line := range lines {
		id, err := ProductID(c.Domain, line.Offer)
		if err != nil {
			return offer.CartQuote{}, err
		}
		products[index] = id
	}
	base := "https://" + c.Domain + "/wp-json/wc/store/v1/cart"
	// The Cart-Token Names the Cart and Spares the Nonce a Browser would Carry.
	_, header, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodGet, Target: base})
	if err != nil {
		return offer.CartQuote{}, err
	}
	token := header.Get("Cart-Token")
	if token == "" {
		return offer.CartQuote{}, fmt.Errorf("%s answered no Cart-Token", c.Domain)
	}
	session := http.Header{"Cart-Token": {token}}
	for index, line := range lines {
		body, _ := json.Marshal(map[string]int{"id": products[index], "quantity": line.Quantity})
		if _, _, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodPost, Target: base + "/add-item", Body: body, Header: session}); err != nil {
			return offer.CartQuote{}, fmt.Errorf("add %s: %w", line.Offer.ID, err)
		}
	}
	body, _ := json.Marshal(map[string]any{"shipping_address": map[string]string{
		"country": address.Country, "state": address.Region, "city": address.City, "postcode": address.Postcode,
	}})
	data, _, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodPost, Target: base + "/update-customer", Body: body, Header: session})
	if err != nil {
		return offer.CartQuote{}, err
	}
	var cart cartReply
	if err := json.Unmarshal(data, &cart); err != nil {
		return offer.CartQuote{}, fmt.Errorf("decode cart %s: %w", c.Domain, err)
	}
	return readQuote(cart, lines, products)
}

func readQuote(cart cartReply, lines []offer.CartLine, products []int) (offer.CartQuote, error) {
	minor := cart.Totals.MinorUnit
	quote := offer.CartQuote{Currency: cart.Totals.Currency, PaymentMethods: cart.PaymentMethods}
	var err error
	if quote.Items, err = formatAmount(cart.Totals.Items, minor); err != nil {
		return offer.CartQuote{}, err
	}
	if quote.Shipping, err = formatAmount(cart.Totals.Shipping, minor); err != nil {
		return offer.CartQuote{}, err
	}
	if quote.Total, err = formatAmount(cart.Totals.Total, minor); err != nil {
		return offer.CartQuote{}, err
	}
	wanted := map[int]int{}
	for index, line := range lines {
		wanted[products[index]] += line.Quantity
	}
	held := map[int]int{}
	for index, line := range lines {
		echo := offer.QuoteLine{OfferID: line.Offer.ID, Quantity: line.Quantity}
		for _, item := range cart.Items {
			if item.ID == products[index] {
				held[item.ID] = item.Quantity
				echo.UnitPrice, _ = formatAmount(item.Prices.Price, item.Prices.MinorUnit)
			}
		}
		quote.Lines = append(quote.Lines, echo)
	}
	// A Product the Cart Dropped or Trimmed is the Store Saying it has Fewer.
	for index, line := range lines {
		id := products[index]
		if count, asked := held[id], wanted[id]; count != asked {
			quote.Notices = append(quote.Notices, fmt.Sprintf("%s: asked %d, cart holds %d", line.Offer.ID, asked, count))
			wanted[id] = count
		}
	}
	for _, pack := range cart.Packages {
		for _, rate := range pack.Rates {
			price, _ := formatAmount(rate.Price, minor)
			quote.ShippingRates = append(quote.ShippingRates, offer.ShippingRate{ID: rate.ID, Name: html.UnescapeString(rate.Name), Price: price, Selected: rate.Selected})
		}
	}
	for _, notice := range cart.Errors {
		quote.Notices = append(quote.Notices, html.UnescapeString(notice.Message))
	}
	return quote, nil
}

// formatAmount Reads a Store API Amount, where Zero is a Real Answer.
func formatAmount(value string, minor int) (string, error) {
	if value == "0" {
		return "0", nil
	}
	return formatPrice(value, minor)
}

// ProductID Reads the Product a Store's own Offer Names. An Aggregator's Offer
// Names its own Record, and a Variable Product Needs a Variation this Catalog
// does not Keep, so both are Refused.
func ProductID(domain string, item offer.Offer) (int, error) {
	if item.Source != domain || item.VariantID == "" {
		return 0, ErrNotQuotable
	}
	id, err := strconv.Atoi(item.VariantID)
	if err != nil || id <= 0 {
		return 0, ErrNotQuotable
	}
	return id, nil
}
