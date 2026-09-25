package stores

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// Quoter Asks a Store's own Cart what the Buyer's Lines would Cost to Ship.
type Quoter struct {
	Sessions woocommerce.SessionSender
	Config   Config
}

// QuoteCart Answers offer.ErrNoQuote for a Store whose Platform has no Cart to Ask.
func (q Quoter) QuoteCart(ctx context.Context, domain string, lines []offer.CartLine, address offer.ShippingAddress) (offer.CartQuote, error) {
	config, found := q.Config.Stores[domain]
	if !found || !config.Enabled || config.Platform != "woocommerce" || q.Sessions == nil {
		return offer.CartQuote{}, offer.ErrNoQuote
	}
	client := woocommerce.Client{Domain: domain, Name: config.Name, Sessions: q.Sessions}
	return client.QuoteCart(ctx, lines, address)
}
