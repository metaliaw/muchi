package search

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// SearchStore Persists Searches under https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml.
type SearchStore interface {
	CreateSearch(context.Context, string, string, CreateInput) (Job, error)
	GetSearch(context.Context, string) (Job, error)
	ListResults(context.Context, string, ResultPage) (Result, error)
	CancelSearch(context.Context, string, string, string) (Job, error)
	ClaimSearchItem(context.Context, string, time.Duration) (Item, error)
	RenewItemLease(context.Context, string, string, time.Duration) error
	CompleteSearchItem(context.Context, Item, []offer.Offer) error
}

// TaskQueue Dispatches Card Work through https://cloud.google.com/tasks/docs/reference/rest.
type TaskQueue interface {
	DispatchSearch(context.Context, Job) error
}

// OfferSource Finds Offers shaped by https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml.
type OfferSource interface {
	FindOffers(context.Context, string) ([]offer.Offer, error)
}

// StockChecker Verifies availability using https://schema.org/availability.
type StockChecker interface {
	CheckStock(context.Context, offer.Offer) (string, error)
}

// OfferCache Reuses Offers under https://www.rfc-editor.org/rfc/rfc9111.html.
type OfferCache interface {
	LoadOffers(context.Context, string) ([]offer.Offer, bool, error)
	SaveOffers(context.Context, string, []offer.Offer, time.Duration) error
}

// HealthStore Reports health through https://github.com/cangrejometralleta/muchi-api/blob/main/openapi.yaml.
type HealthStore interface {
	CheckHealth(context.Context) error
	ListSourceHealth(context.Context) ([]SourceHealth, error)
}
