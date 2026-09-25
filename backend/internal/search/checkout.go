package search

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"

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
	// Quote is the Store's own Cart Answer, Present only when the Caller Sent
	// an Address and the Store can be Asked. QuoteError Says why one Failed.
	Quote      *offer.CartQuote `json:"quote,omitempty"`
	QuoteError string           `json:"quote_error,omitempty"`
}

// CheckoutLinks Hands the Buyer one Way into each Store's Checkout.
//
// Nothing is Bought: the Links only Carry the Cart the Buyer Chose, and the
// Buyer Pays at the Store. Offers are Named by the Id
// the Search gave them, for the same Reason CheckOfferStock Asks for it.
//
// With an Address, each Store that can be Asked Fills a Cart of its own and
// Answers its Total with Shipping. That Cart Holds no Stock and no Order.
func (s Service) CheckoutLinks(ctx context.Context, id string, requests []CartRequest, address *offer.ShippingAddress) ([]StoreCheckout, error) {
	if id == "" || len(requests) == 0 || len(requests) > maxCheckedOffers {
		return nil, ErrInvalid
	}
	for _, request := range requests {
		if request.OfferID == "" || request.Quantity <= 0 || request.Quantity > maxCheckoutUnits {
			return nil, ErrInvalid
		}
	}
	if address != nil && strings.TrimSpace(address.Country) == "" {
		return nil, ErrInvalid
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
	checkouts := make([]StoreCheckout, len(domains))
	var group sync.WaitGroup
	gate := make(chan struct{}, maxConcurrentSources)
	for index, domain := range domains {
		checkouts[index] = s.planCheckout(domain, groups[domain])
		if address == nil || s.Quotes == nil {
			continue
		}
		group.Add(1)
		go func(checkout *StoreCheckout, lines []offer.CartLine) {
			defer group.Done()
			gate <- struct{}{}
			defer func() { <-gate }()
			s.quoteCheckout(ctx, checkout, lines, *address)
		}(&checkouts[index], groups[domain])
	}
	group.Wait()
	return checkouts, nil
}

// quoteCheckout Keeps a Store Failure inside the Answer, as readOfferStock does:
// the Links Stand whether or not the Store Priced its Shipping.
func (s Service) quoteCheckout(ctx context.Context, checkout *StoreCheckout, lines []offer.CartLine, address offer.ShippingAddress) {
	quote, err := s.Quotes.QuoteCart(ctx, checkout.Domain, lines, address)
	switch {
	case errors.Is(err, offer.ErrNoQuote):
	case err != nil:
		checkout.QuoteError = "the store did not answer its cart"
	default:
		checkout.Quote = &quote
	}
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
