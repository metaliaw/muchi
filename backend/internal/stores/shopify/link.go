package shopify

import (
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// CartLink Names every Variant in one Permalink; Shopify Fills the Cart and
// Opens Checkout from it directly:
// https://help.shopify.com/en/manual/products/details/cart-permalink
// It Answers false when a Line cannot Name its Variant.
func (c Client) CartLink(lines []model.CartLine) (string, bool) {
	quantities := map[string]int{}
	var order []string
	for _, line := range lines {
		variant := VariantID(c.Domain, line.Offer)
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
	return "https://" + c.Domain + "/cart/" + strings.Join(parts, ","), true
}
