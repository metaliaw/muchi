// Package commerce Opens the Platform Client behind each Enabled Store and
// Answers for what the Stores can Do. It Reads the Directory and Switches on a
// Platform in one Place; the Clients Live in the Packages Beside it.
package commerce

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
	"github.com/cangrejometralleta/muchi-api/internal/stores/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/stores/prestashop"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

// Shops Opens the Platform Client behind each Enabled Store and Answers for
// what the Stores can Do: Stock, Quotes, Orders, Checkout Links and Offers. It
// is the one Place that Switches on a Platform; every Answer Asks the Client
// for the Capability it Needs, and a Client without it Answers Not Supported.
type Shops struct {
	Config   directory.Config
	Fetcher  SourceFetcher
	Sessions woocommerce.SessionSender
	// BuyerName and BuyerEmail Name the Buyer every Order is Placed as, Read
	// from MUCHI_ORDER_BUYER_NAME and MUCHI_ORDER_BUYER_EMAIL. Empty Falls
	// back to Muchi's own reserved Placeholder, which no Platform Checks out
	// with.
	BuyerName  string
	BuyerEmail string
	// Lists Answers for the Stores that Publish an Inventory instead of a
	// Storefront. Their Offers all Share one URL, so the Host of an Offer Names
	// no Store and the Question would Die as "unknown".
	Lists StockReader
}

// Shop Is one Enabled Store: its Configuration and the Client its Platform
// Speaks through. Client Stays nil for a Platform that has none.
type Shop struct {
	Config directory.StoreConfig
	Client any
}

// OfferFinder Finds Offers in one Store, as the Search Reads them.
type OfferFinder interface {
	FindOffers(context.Context, model.CardQuery) ([]model.Offer, error)
	SourceName() string
}

// cartQuoter Asks a Store's own Cart what Lines would Cost with Shipping.
type cartQuoter interface {
	QuoteCart(context.Context, []model.CartLine, model.ShippingAddress) (model.CartQuote, error)
}

// orderPlacer Checks a Cart out at its Store, for real.
type orderPlacer interface {
	PlaceOrder(context.Context, []model.CartLine, model.ShippingAddress) (model.Order, error)
}

// cartLinker Builds one URL that Fills a Store's Cart and Opens its Checkout.
type cartLinker interface {
	CartLink([]model.CartLine) (string, bool)
}

// Shop Answers the Enabled Store Behind a Domain, or false for one that is
// Unknown or Disabled: nothing Asks a Store that has not Opted in.
func (s Shops) Shop(domain string) (Shop, bool) {
	config, found := s.Config.Stores[domain]
	if !found || !config.Enabled {
		return Shop{}, false
	}
	return Shop{Config: config, Client: s.openClient(domain, config)}, true
}

// Finder Answers the Client that Searches this Store's own Catalog.
func (s Shops) Finder(domain string) (OfferFinder, bool) {
	shop, found := s.Shop(domain)
	if !found {
		return nil, false
	}
	finder, can := shop.Client.(OfferFinder)
	return finder, can
}

// openClient Builds the Client for one Platform, with every Dependency any of
// its Capabilities Needs. This is the only Switch on a Platform Name.
func (s Shops) openClient(domain string, config directory.StoreConfig) any {
	switch config.Platform {
	case "jumpseller":
		return jumpseller.Client{Fetcher: s.Fetcher, Domain: domain, Name: config.Name}
	case "shopify":
		return shopify.Client{Fetcher: s.Fetcher, Domain: domain, Name: config.Name, Sessions: s.Sessions}
	case "woocommerce":
		return woocommerce.Client{
			Fetcher: s.Fetcher, Domain: domain, Name: config.Name, Sessions: s.Sessions,
			BuyerName: s.BuyerName, BuyerEmail: s.BuyerEmail,
		}
	case "prestashop":
		return prestashop.Client{Fetcher: s.Fetcher, Domain: domain, Name: config.Name, SearchPath: "/busqueda"}
	default:
		return nil
	}
}
