package stores

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// CheckoutLink Builds one URL that Fills a Store's Cart with every Line and
// Opens its Checkout. It Answers false when the Store's Platform has no such
// Link, or when a Line cannot Name its Variant, so the Caller Falls back to the
// Product Pages instead of Sending the Buyer to a Cart Missing a Card.
//
// Only Shopify Qualifies today: https://help.shopify.com/en/manual/products/details/cart-permalink
func (c Config) CheckoutLink(domain string, lines []offer.CartLine) (string, bool) {
	config, found := c.Stores[domain]
	if !found || !config.Enabled || config.Platform != "shopify" || len(lines) == 0 {
		return "", false
	}
	quantities := map[string]int{}
	var order []string
	for _, line := range lines {
		variant := shopifyVariant(domain, line.Offer)
		if variant == "" || line.Quantity <= 0 {
			return "", false
		}
		if _, seen := quantities[variant]; !seen {
			order = append(order, variant)
		}
		quantities[variant] += line.Quantity
	}
	parts := make([]string, len(order))
	for index, variant := range order {
		parts[index] = variant + ":" + strconv.Itoa(quantities[variant])
	}
	return "https://" + domain + "/cart/" + strings.Join(parts, ","), true
}

// shopifyVariant Trusts the Variant the Product URL Carries, and the Offer's own
// Variant only when the Store Itself Sent it: an Aggregator's Key Names its
// own Record, not a Shopify Variant.
func shopifyVariant(domain string, item offer.Offer) string {
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
