package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// SessionSender Makes the Calls a Cart Lives in.
type SessionSender interface {
	SendSession(context.Context, string, source.SessionRequest) ([]byte, http.Header, error)
}

type cartReply struct {
	Currency string `json:"currency"`
	Total    int64  `json:"total_price"`
	Items    []struct {
		ID       int64 `json:"variant_id"`
		Quantity int   `json:"quantity"`
		Price    int64 `json:"final_price"`
	} `json:"items"`
}

type ratesReply struct {
	Rates []struct {
		Code  string `json:"code"`
		Name  string `json:"name"`
		Price string `json:"price"`
	} `json:"shipping_rates"`
}

// QuoteCart Fills a Fresh Ajax Cart with the Lines and Asks the Store what it
// would Charge to Ship it. The Cart Lives in a Cookie this Call Keeps to
// itself; it Holds no Stock and Creates no Checkout. The Shipping Total is the
// Cheapest Rate, since a Storefront Cart has no Rate Selected yet.
//
// https://shopify.dev/docs/api/ajax/reference/cart
func (c Client) QuoteCart(ctx context.Context, lines []offer.CartLine, address offer.ShippingAddress) (offer.CartQuote, error) {
	if c.Sessions == nil || len(lines) == 0 {
		return offer.CartQuote{}, offer.ErrNoQuote
	}
	type item struct {
		ID       int64 `json:"id"`
		Quantity int   `json:"quantity"`
	}
	items := make([]item, len(lines))
	variants := make([]int64, len(lines))
	for index, line := range lines {
		id, err := strconv.ParseInt(VariantID(c.Domain, line.Offer), 10, 64)
		if err != nil {
			return offer.CartQuote{}, offer.ErrNoQuote
		}
		items[index], variants[index] = item{ID: id, Quantity: line.Quantity}, id
	}
	base := "https://" + c.Domain
	body, _ := json.Marshal(map[string]any{"items": items})
	_, header, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodPost, Target: base + "/cart/add.js", Body: body})
	if err != nil {
		return offer.CartQuote{}, err
	}
	session := http.Header{"Cookie": {readCookies(header)}}
	data, _, err := c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodGet, Target: base + "/cart.js", Header: session})
	if err != nil {
		return offer.CartQuote{}, err
	}
	var cart cartReply
	if err := json.Unmarshal(data, &cart); err != nil {
		return offer.CartQuote{}, fmt.Errorf("decode cart %s: %w", c.Domain, err)
	}
	query := url.Values{
		"shipping_address[country]":  {address.Country},
		"shipping_address[province]": {address.Region},
		"shipping_address[zip]":      {address.Postcode},
	}
	data, _, err = c.Sessions.SendSession(ctx, c.Domain, source.SessionRequest{Method: http.MethodGet, Target: base + "/cart/shipping_rates.json?" + query.Encode(), Header: session})
	if err != nil {
		return offer.CartQuote{}, err
	}
	var rates ratesReply
	if err := json.Unmarshal(data, &rates); err != nil {
		return offer.CartQuote{}, fmt.Errorf("decode shipping rates %s: %w", c.Domain, err)
	}
	return readQuote(cart, rates, lines, variants), nil
}

func readQuote(cart cartReply, rates ratesReply, lines []offer.CartLine, variants []int64) offer.CartQuote {
	quote := offer.CartQuote{Currency: cart.Currency, Items: formatPrice(cart.Total), PaymentMethods: []string{}}
	wanted, held := map[int64]int{}, map[int64]int{}
	for index, line := range lines {
		wanted[variants[index]] += line.Quantity
	}
	for _, entry := range cart.Items {
		held[entry.ID] += entry.Quantity
	}
	for index, line := range lines {
		echo := offer.QuoteLine{OfferID: line.Offer.ID, Quantity: line.Quantity}
		for _, entry := range cart.Items {
			if entry.ID == variants[index] {
				echo.UnitPrice = formatPrice(entry.Price)
			}
		}
		quote.Lines = append(quote.Lines, echo)
	}
	for index, line := range lines {
		id := variants[index]
		if held[id] != wanted[id] {
			quote.Notices = append(quote.Notices, fmt.Sprintf("%s: asked %d, cart holds %d", line.Offer.ID, wanted[id], held[id]))
			wanted[id] = held[id]
		}
	}
	cheapest := int64(-1)
	for _, rate := range rates.Rates {
		amount, err := strconv.ParseFloat(rate.Price, 64)
		if err != nil || amount < 0 {
			continue
		}
		cents := int64(math.Round(amount * 100))
		quote.ShippingRates = append(quote.ShippingRates, offer.ShippingRate{ID: rate.Code, Name: rate.Name, Price: formatPrice(cents)})
		if cheapest < 0 || cents < cheapest {
			cheapest = cents
		}
	}
	if cheapest < 0 {
		quote.Notices = append(quote.Notices, "the store offered no shipping rate for this address")
		cheapest = 0
	}
	quote.Shipping, quote.Total = formatPrice(cheapest), formatPrice(cart.Total+cheapest)
	return quote
}

// readCookies Replays what the Store Set, so the next Call Reads the same Cart.
func readCookies(header http.Header) string {
	var pairs []string
	for _, line := range header.Values("Set-Cookie") {
		if pair, _, _ := strings.Cut(line, ";"); strings.Contains(pair, "=") {
			pairs = append(pairs, strings.TrimSpace(pair))
		}
	}
	return strings.Join(pairs, "; ")
}

// VariantID Trusts the Variant the Product URL Carries, and the Offer's own
// Variant only when the Store Itself Sent it: an Aggregator's Key Names its
// own Record, not a Shopify Variant. It Answers "" when Neither Qualifies.
func VariantID(domain string, item offer.Offer) string {
	variant := ""
	if link, err := url.Parse(item.URL); err == nil && link.Host == domain {
		variant = link.Query().Get("variant")
	}
	if variant == "" && item.Source == domain {
		variant = item.VariantID
	}
	if id, err := strconv.ParseInt(variant, 10, 64); err != nil || id <= 0 {
		return ""
	}
	return variant
}
