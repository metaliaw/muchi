package woocommerce

import (
	"strconv"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// CartLink Names one Product by `add-to-cart`: the only Link WooCommerce
// Answers without its Store API building a real Order. A second Line has no
// such Link, so the Caller Falls back to the Product Pages instead of
// Dropping a Card from the Cart the Buyer Chose.
func (c Client) CartLink(lines []model.CartLine) (string, bool) {
	if len(lines) != 1 || lines[0].Quantity <= 0 {
		return "", false
	}
	id, err := ProductID(c.Domain, lines[0].Offer)
	if err != nil {
		return "", false
	}
	return "https://" + c.Domain + "/?add-to-cart=" + strconv.Itoa(id) + "&quantity=" + strconv.Itoa(lines[0].Quantity), true
}
