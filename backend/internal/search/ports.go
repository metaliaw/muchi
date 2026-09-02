package search

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type SearchStore interface {
	CreateSearch(context.Context, string, string, CreateInput) (Job, error)
	GetSearch(context.Context, string) (Job, error)
	ListResults(context.Context, string) (Result, error)
	CancelSearch(context.Context, string, string, string) (Job, error)
	ClaimSearchItem(context.Context, string, time.Duration) (Item, error)
	RenewItemLease(context.Context, string, string, time.Duration) error
	CompleteSearchItem(context.Context, Item, []offer.Offer) error
}

type OfferSource interface {
	FindOffers(context.Context, string) ([]offer.Offer, error)
}

type StockChecker interface {
	CheckStock(context.Context, offer.Offer) (string, error)
}

type OfferCache interface {
	LoadOffers(context.Context, string) ([]offer.Offer, bool, error)
	SaveOffers(context.Context, string, []offer.Offer, time.Duration) error
}

type TrafficGate interface {
	AwaitSource(context.Context, string) error
	RecordSource(context.Context, string, time.Duration, error) error
}

type HealthStore interface {
	CheckHealth(context.Context) error
	ListSourceHealth(context.Context) ([]SourceHealth, error)
}
