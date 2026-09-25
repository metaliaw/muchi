package search

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// SearchRepository Stores searches for the service.
type SearchRepository interface {
	CreateSearch(context.Context, string, string, model.CreateInput) (model.Job, error)
	GetSearch(context.Context, string) (model.Job, error)
	ListResults(context.Context, string, model.ResultPage) (model.Result, error)
	CancelSearch(context.Context, string, string, string) (model.Job, error)
}

// SearchItemRepository Claims and completes the worker's card items.
type SearchItemRepository interface {
	ClaimSearchItem(context.Context, string, time.Duration) (model.Item, error)
	RenewItemLease(context.Context, string, string, time.Duration) error
	CompleteSearchItem(context.Context, model.Item, []model.Offer) error
}
