package stores

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// Quoter Asks a Store's own Cart what the Buyer's Lines would Cost to Ship.
type Quoter struct {
	Sessions woocommerce.SessionSender
	Config   Config
}

// QuoteCart Answers offer.ErrNoQuote for a Store whose Platform has no Cart to Ask.
// The Address Arrives in Names and Leaves in each Platform's Codes: WooCommerce
// Reads "CL-RM", Shopify Reads "RM".
func (q Quoter) QuoteCart(ctx context.Context, domain string, lines []offer.CartLine, address offer.ShippingAddress) (offer.CartQuote, error) {
	config, found := q.Config.Stores[domain]
	if !found || !config.Enabled || q.Sessions == nil {
		return offer.CartQuote{}, offer.ErrNoQuote
	}
	address = resolveAddress(address)
	switch config.Platform {
	case "woocommerce":
		if address.Country == "CL" && len(address.Region) == 2 {
			address.Region = "CL-" + address.Region
		}
		return woocommerce.Client{Domain: domain, Name: config.Name, Sessions: q.Sessions}.QuoteCart(ctx, lines, address)
	case "shopify":
		return shopify.Client{Domain: domain, Name: config.Name, Sessions: q.Sessions}.QuoteCart(ctx, lines, address)
	default:
		return offer.CartQuote{}, offer.ErrNoQuote
	}
}
