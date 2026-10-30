package commerce

import "github.com/cangrejometralleta/muchi-api/internal/model"

// CheckoutLink Builds one URL that Fills a Store's Cart with every Line and
// Opens its Checkout. It Answers false when the Store's Platform has no such
// Link, or when a Line cannot Name its Variant, so the Caller Falls back to the
// Product Pages instead of Sending the Buyer to a Cart Missing a Card.
func (s Shops) CheckoutLink(domain string, lines []model.CartLine) (string, bool) {
	shop, found := s.Shop(domain)
	if !found || len(lines) == 0 {
		return "", false
	}
	linker, can := shop.Client.(cartLinker)
	if !can {
		return "", false
	}
	return linker.CartLink(lines)
}
