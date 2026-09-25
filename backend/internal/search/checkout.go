package search

import (
	"context"
	"net/url"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// maxCheckoutUnits Caps the Units of one Line. A Deck Asks for four of a Card;
// a Number far above that is a Typo, not a Purchase.
const maxCheckoutUnits = 99

// CartRequest Asks for some Units of an Offer this Search Found.
type CartRequest struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
}

// CheckoutLine Echoes one Line with the Page it can be Bought from.
type CheckoutLine struct {
	OfferID  string `json:"offer_id"`
	Quantity int    `json:"quantity"`
	URL      string `json:"url"`
}

// StoreCheckout Groups the Lines one Store will Sell. Mode "cart" Carries a URL
// that Fills the Cart and Opens Checkout; Mode "product_pages" Leaves the Buyer
// the Page of each Line.
type StoreCheckout struct {
	Store  string         `json:"store"`
	Domain string         `json:"domain"`
	Mode   string         `json:"mode"`
	URL    string         `json:"url,omitempty"`
	Lines  []CheckoutLine `json:"lines"`
}

// CheckoutLinks Hands the Buyer one Way into each Store's Checkout.
//
// Nothing is Bought and no Cart is Touched here: the Links only Carry the Cart
// the Buyer Chose, and the Buyer Pays at the Store. Offers are Named by the Id
// the Search gave them, for the same Reason CheckOfferStock Asks for it.
func (s Service) CheckoutLinks(ctx context.Context, id string, requests []CartRequest) ([]StoreCheckout, error) {
	if id == "" || len(requests) == 0 || len(requests) > maxCheckedOffers {
		return nil, ErrInvalid
	}
	for _, request := range requests {
		if request.OfferID == "" || request.Quantity <= 0 || request.Quantity > maxCheckoutUnits {
			return nil, ErrInvalid
		}
	}
	found, err := s.collectSearchOffers(ctx, id)
	if err != nil {
		return nil, err
	}
	var domains []string
	groups := map[string][]offer.CartLine{}
	for _, request := range requests {
		item, known := found[request.OfferID]
		if !known {
			return nil, ErrNotFound
		}
		domain := ""
		if link, err := url.Parse(item.URL); err == nil {
			domain = link.Host
		}
		if _, seen := groups[domain]; !seen {
			domains = append(domains, domain)
		}
		groups[domain] = append(groups[domain], offer.CartLine{Offer: item, Quantity: request.Quantity})
	}
	checkouts := make([]StoreCheckout, 0, len(domains))
	for _, domain := range domains {
		checkouts = append(checkouts, s.planCheckout(domain, groups[domain]))
	}
	return checkouts, nil
}

func (s Service) planCheckout(domain string, lines []offer.CartLine) StoreCheckout {
	checkout := StoreCheckout{Store: lines[0].Offer.Store, Domain: domain, Mode: "product_pages"}
	for _, line := range lines {
		checkout.Lines = append(checkout.Lines, CheckoutLine{OfferID: line.Offer.ID, Quantity: line.Quantity, URL: line.Offer.URL})
	}
	if s.Checkouts == nil {
		return checkout
	}
	if link, ok := s.Checkouts.CheckoutLink(domain, lines); ok {
		checkout.Mode, checkout.URL = "cart", link
	}
	return checkout
}
