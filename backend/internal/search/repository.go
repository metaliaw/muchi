package search

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// SearchRepository Stores searches for the service.
type SearchRepository interface {
	CreateSearch(context.Context, string, string, CreateInput) (Job, error)
	GetSearch(context.Context, string) (Job, error)
	ListResults(context.Context, string, ResultPage) (Result, error)
	CancelSearch(context.Context, string, string, string) (Job, error)
}

// SearchItemRepository Claims and completes the worker's card items.
type SearchItemRepository interface {
	ClaimSearchItem(context.Context, string, time.Duration) (Item, error)
	RenewItemLease(context.Context, string, string, time.Duration) error
	CompleteSearchItem(context.Context, Item, []offer.Offer) error
}
