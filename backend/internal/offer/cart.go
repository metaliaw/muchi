package offer

import "errors"

// CartLine Asks for some Units of one Offer.
type CartLine struct {
	Offer    Offer
	Quantity int
}

// ShippingAddress Names where a Cart would Travel, as much as a Store Needs to
// Price its Shipping. Region is the Store's own State Code, such as "CL-RM".
type ShippingAddress struct {
	Country  string `json:"country"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
	Postcode string `json:"postcode,omitempty"`
}

// CartQuote is what a Store's own Cart Answered, Prices as the Offers Write them.
// Nothing is Reserved: the Store Keeps Selling while the Buyer Decides.
type CartQuote struct {
	Currency       string         `json:"currency"`
	Items          string         `json:"items_total"`
	Shipping       string         `json:"shipping_total"`
	Total          string         `json:"total"`
	Lines          []QuoteLine    `json:"lines"`
	ShippingRates  []ShippingRate `json:"shipping_rates"`
	PaymentMethods []string       `json:"payment_methods"`
	Notices        []string       `json:"notices,omitempty"`
}

// QuoteLine Echoes one Line with the Price the Cart Charged for it.
type QuoteLine struct {
	OfferID   string `json:"offer_id"`
	Quantity  int    `json:"quantity"`
	UnitPrice string `json:"unit_price"`
}

// ShippingRate is one way the Store Offers to Ship the Cart.
type ShippingRate struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Price    string `json:"price"`
	Selected bool   `json:"selected"`
}

// ErrNoQuote Answers a Cart no Store Cart can Price: the Platform has no Cart
// to Ask, or a Line cannot Name its Product there.
var ErrNoQuote = errors.New("cart cannot be quoted at this store")
