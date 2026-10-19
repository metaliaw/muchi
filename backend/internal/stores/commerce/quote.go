package commerce

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// QuoteCart Asks a Store's own Cart what the Buyer's Lines would Cost to Ship.
// It Answers model.ErrNoQuote for a Store whose Platform has no Cart to Ask.
func (s Shops) QuoteCart(ctx context.Context, domain string, lines []model.CartLine, address model.ShippingAddress) (model.CartQuote, error) {
	shop, found := s.Shop(domain)
	if !found || s.Sessions == nil {
		return model.CartQuote{}, model.ErrNoQuote
	}
	quoter, can := shop.Client.(cartQuoter)
	if !can {
		return model.CartQuote{}, model.ErrNoQuote
	}
	return quoter.QuoteCart(ctx, lines, prepareAddress(shop.Config.Platform, address))
}
