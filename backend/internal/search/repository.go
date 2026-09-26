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

// OrderRepository Persists a placed Order and Moves it between Statuses. A
// second Create with the same idempotency Key and Payload Answers the same
// Order, never a new one.
type OrderRepository interface {
	CreateOrder(ctx context.Context, key, hash string, order model.Order) (model.Order, error)
	GetOrder(ctx context.Context, id string) (model.Order, error)
	MoveOrderStatus(ctx context.Context, id string, from, to model.OrderStatus) (model.Order, error)
}
