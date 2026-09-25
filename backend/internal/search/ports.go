package search

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TaskQueue Dispatches Card Work through https://cloud.google.com/tasks/docs/reference/rest.
type TaskQueue interface {
	DispatchSearch(context.Context, model.Job) error
}

// Provider Searches one Game through its configured external Catalog.
type Provider interface {
	Search(context.Context, model.CardQuery) ([]model.Offer, error)
	SourceName() string
}

// OfferSource Finds Offers in one directly configured Store.
type OfferSource interface {
	FindOffers(context.Context, model.CardQuery) ([]model.Offer, error)
	// SourceName Lets a Failure Say which Source Failed.
	SourceName() string
}

// PrintLibrary Lists the Printings of a Card, each with its Image.
type PrintLibrary interface {
	CardPrints(context.Context, string) ([]cardmetadata.Print, error)
}

// SetLibrary Lists the Sets one Game Has Released.
type SetLibrary interface {
	GameSets(context.Context) ([]string, error)
}

// StockChecker Verifies availability using https://schema.org/availability.
type StockChecker interface {
	CheckStock(context.Context, model.Offer) (model.StockReading, error)
}

// CheckoutLinker Builds a URL that Fills one Store's Cart and Opens its Checkout.
type CheckoutLinker interface {
	CheckoutLink(domain string, lines []model.CartLine) (string, bool)
}

// CartQuoter Asks a Store's own Cart what the Lines would Cost with Shipping.
// A Store it cannot Ask Answers offer.ErrNoQuote.
type CartQuoter interface {
	QuoteCart(ctx context.Context, domain string, lines []model.CartLine, address model.ShippingAddress) (model.CartQuote, error)
}

// OfferCache Reuses Offers under https://www.rfc-editor.org/rfc/rfc9111.html.
type OfferCache interface {
	LoadOffers(context.Context, string) ([]model.Offer, bool, error)
	SaveOffers(context.Context, string, []model.Offer, time.Duration) error
}

// HealthStore Reports health through https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml.
type HealthStore interface {
	CheckHealth(context.Context) error
	ListSourceHealth(context.Context) ([]model.SourceHealth, error)
}
