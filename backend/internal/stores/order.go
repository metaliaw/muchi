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
	// BuyerName and BuyerEmail Name the Buyer every Order is Placed as, Read
	// from MUCHI_ORDER_BUYER_NAME and MUCHI_ORDER_BUYER_EMAIL. Empty Falls
	// back to Muchi's own reserved Placeholder, which no Platform Checks out
	// with.
	BuyerName  string
	BuyerEmail string
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
	// resolveAddress Turns "Chile"/"Región Metropolitana" into "CL"/"RM" the
	// way Quoter Reads them; WooCommerce's own State Code Needs the further
	// "CL-" Prefix Quoter Adds too — Checkout Deserves the same Address
	// Quoting already Agreed on, never a raw Buyer Spelling the Store API may
	// Refuse or Silently Mis-rate.
	address = resolveAddress(address)
	if address.Country == "CL" && len(address.Region) == 2 {
		address.Region = "CL-" + address.Region
	}
	client := woocommerce.Client{
		Domain: domain, Name: config.Name, Sessions: o.Sessions,
		BuyerName: o.BuyerName, BuyerEmail: o.BuyerEmail,
	}
	return client.PlaceOrder(ctx, lines, address)
}
