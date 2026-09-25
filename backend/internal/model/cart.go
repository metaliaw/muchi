package model

import "errors"

// CartLine Asks for some Units of one Offer.
type CartLine struct {
	Offer    Offer
	Quantity int
}

// ShippingAddress Names where a Cart would Travel, as much as a Store Needs to
// Price its Shipping. Region is the Store's own State Code, such as "CL-RM".
type ShippingAddress struct {
	Country  string
	Region   string
	City     string
	Postcode string
}

// CartQuote is what a Store's own Cart Answered, Prices as the Offers Write them.
// Nothing is Reserved: the Store Keeps Selling while the Buyer Decides.
type CartQuote struct {
	Currency       string
	Items          string
	Shipping       string
	Total          string
	Lines          []QuoteLine
	ShippingRates  []ShippingRate
	PaymentMethods []string
	Notices        []string
}

// QuoteLine Echoes one Line with the Price the Cart Charged for it.
type QuoteLine struct {
	OfferID   string
	Quantity  int
	UnitPrice string
}

// ShippingRate is one way the Store Offers to Ship the Cart.
type ShippingRate struct {
	ID       string
	Name     string
	Price    string
	Selected bool
}

// ErrNoQuote Answers a Cart no Store Cart can Price: the Platform has no Cart
// to Ask, or a Line cannot Name its Product there.
var ErrNoQuote = errors.New("cart cannot be quoted at this store")

// CartRequest Asks for some Units of an Offer this Search Found.
type CartRequest struct {
	OfferID  string
	Quantity int
}

// CheckoutLine Echoes one Line with the Page it can be Bought from.
type CheckoutLine struct {
	OfferID  string
	Quantity int
	URL      string
}

// StoreCheckout Groups the Lines one Store will Sell. Mode "cart" Carries a URL
// that Fills the Cart and Opens Checkout; Mode "product_pages" Leaves the Buyer
// the Page of each Line.
type StoreCheckout struct {
	Store  string
	Domain string
	Mode   string
	URL    string
	Lines  []CheckoutLine
	// Quote is the Store's own Cart Answer, Present only when the Caller Sent
	// an Address and the Store can be Asked. QuoteError Says why one Failed.
	Quote      *CartQuote
	QuoteError string
}

// OfferStock Names the Offer the Reading Belongs to.
type OfferStock struct {
	ID string
	StockReading
}
