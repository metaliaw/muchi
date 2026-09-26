package stores

import (
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// CheckoutLink Builds one URL that Fills a Store's Cart with every Line and
// Opens its Checkout. It Answers false when the Store's Platform has no such
// Link, or when a Line cannot Name its Variant, so the Caller Falls back to the
// Product Pages instead of Sending the Buyer to a Cart Missing a Card.
func (c Config) CheckoutLink(domain string, lines []model.CartLine) (string, bool) {
	config, found := c.Stores[domain]
	if !found || !config.Enabled || len(lines) == 0 {
		return "", false
	}
	switch config.Platform {
	case "shopify":
		return shopifyCartLink(domain, lines)
	case "woocommerce":
		return woocommerceCartLink(domain, lines)
	default:
		return "", false
	}
}

// shopifyCartLink Names every Variant in one Permalink; Shopify Fills the Cart
// and Opens Checkout from it directly:
// https://help.shopify.com/en/manual/products/details/cart-permalink
func shopifyCartLink(domain string, lines []model.CartLine) (string, bool) {
	quantities := map[string]int{}
	var order []string
	for _, line := range lines {
		variant := shopify.VariantID(domain, line.Offer)
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

// woocommerceCartLink Names one Product by `add-to-cart`: the only Link
// WooCommerce Answers without its Store API building a real Order. A second
// Line has no such Link, so the Caller Falls back to the Product Pages
// instead of Dropping a Card from the Cart the Buyer Chose.
func woocommerceCartLink(domain string, lines []model.CartLine) (string, bool) {
	if len(lines) != 1 || lines[0].Quantity <= 0 {
		return "", false
	}
	id, err := woocommerce.ProductID(domain, lines[0].Offer)
	if err != nil {
		return "", false
	}
	return "https://" + domain + "/?add-to-cart=" + strconv.Itoa(id) + "&quantity=" + strconv.Itoa(lines[0].Quantity), true
}
