package commerce

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// PlaceOrder Asks a Store's own Checkout to Place a real Order. Unlike
// QuoteCart, it Never Reads a Store that has not Enabled it: Placing is
// Irreversible in a way Quoting never is. It Answers model.ErrOrderNotSupported
// for a Store whose Platform has no Order to Place; the Caller Falls back to a
// Link instead.
func (s Shops) PlaceOrder(ctx context.Context, domain string, lines []model.CartLine, address model.ShippingAddress) (model.Order, error) {
	shop, found := s.Shop(domain)
	if !found || s.Sessions == nil {
		return model.Order{}, model.ErrOrderNotSupported
	}
	placer, can := shop.Client.(orderPlacer)
	if !can {
		return model.Order{}, model.ErrOrderNotSupported
	}
	return placer.PlaceOrder(ctx, lines, prepareAddress(shop.Config.Platform, address))
}
