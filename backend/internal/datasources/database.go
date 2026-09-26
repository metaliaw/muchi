// Package datasources Defines provider-neutral persistence contracts.
package datasources

import (
	"context"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// Database Exposes persistence operations without naming a storage provider.
type Database interface {
	CreateSearch(context.Context, string, string, model.CreateInput) (model.Job, error)
	GetSearch(context.Context, string) (model.Job, error)
	ListResults(context.Context, string, model.ResultPage) (model.Result, error)
	CancelSearch(context.Context, string, string, string) (model.Job, error)
	ClaimSearchItem(context.Context, string, time.Duration) (model.Item, error)
	RenewItemLease(context.Context, string, string, time.Duration) error
	CompleteSearchItem(context.Context, model.Item, []model.Offer) error
	CountWaitingItems(context.Context, int) (int, error)
	LoadOffers(context.Context, string) ([]model.Offer, bool, error)
	SaveOffers(context.Context, string, []model.Offer, time.Duration) error
	DropOffers(context.Context, string) error
	CheckHealth(context.Context) error
	ListSourceHealth(context.Context) ([]model.SourceHealth, error)
	AwaitSource(context.Context, string) error
	RecordSource(context.Context, string, time.Duration, error) error
	CreateOrder(context.Context, string, string, model.Order) (model.Order, error)
	GetOrder(context.Context, string) (model.Order, error)
	MoveOrderStatus(context.Context, string, model.OrderStatus, model.OrderStatus) (model.Order, error)
	CloseStore() error
}
