package stores

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// Orderer Asks a Store's own Checkout to Place a real Order. Unlike Quoter,
// it Never Reads a Store that has not Enabled it: Placing is Irreversible in
// a way Quoting never is.
type Orderer struct {
	Sessions woocommerce.SessionSender
	Config   Config
}

// PlaceOrder Answers model.ErrOrderNotSupported for a Store whose Platform
// has no Order to Place; the Caller Falls back to a Link instead.
func (o Orderer) PlaceOrder(ctx context.Context, domain string, lines []model.CartLine, address model.ShippingAddress) (model.Order, error) {
	config, found := o.Config.Stores[domain]
	if !found || !config.Enabled || o.Sessions == nil {
		return model.Order{}, model.ErrOrderNotSupported
	}
	if config.Platform != "woocommerce" {
		return model.Order{}, model.ErrOrderNotSupported
	}
	return woocommerce.Client{Domain: domain, Name: config.Name, Sessions: o.Sessions}.PlaceOrder(ctx, lines, address)
}
