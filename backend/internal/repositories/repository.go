// Package repositories Adapts datasource operations to the application's ports.
package repositories

import (
	"context"
	"log/slog"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/datasources"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// Repository Implements the Search Ports on top of the configured database.
type Repository struct {
	database datasources.Database
}

// New Adapts a Generic Database to the Search and Worker Ports.
func New(database datasources.Database) *Repository { return &Repository{database: database} }

func (r *Repository) TellStore(logger *slog.Logger) {
	if teller, ok := r.database.(interface{ TellStore(*slog.Logger) }); ok {
		teller.TellStore(logger)
	}
}

func (r *Repository) CreateSearch(ctx context.Context, key, hash string, input model.CreateInput) (model.Job, error) {
	return r.database.CreateSearch(ctx, key, hash, input)
}

func (r *Repository) GetSearch(ctx context.Context, id string) (model.Job, error) {
	return r.database.GetSearch(ctx, id)
}

func (r *Repository) ListResults(ctx context.Context, id string, page model.ResultPage) (model.Result, error) {
	return r.database.ListResults(ctx, id, page)
}

func (r *Repository) CancelSearch(ctx context.Context, id, key, hash string) (model.Job, error) {
	return r.database.CancelSearch(ctx, id, key, hash)
}

func (r *Repository) ClaimSearchItem(ctx context.Context, owner string, lease time.Duration) (model.Item, error) {
	return r.database.ClaimSearchItem(ctx, owner, lease)
}

func (r *Repository) RenewItemLease(ctx context.Context, id, owner string, lease time.Duration) error {
	return r.database.RenewItemLease(ctx, id, owner, lease)
}

func (r *Repository) CompleteSearchItem(ctx context.Context, item model.Item, offers []model.Offer) error {
	return r.database.CompleteSearchItem(ctx, item, offers)
}

func (r *Repository) CountWaitingItems(ctx context.Context, limit int) (int, error) {
	return r.database.CountWaitingItems(ctx, limit)
}

func (r *Repository) LoadOffers(ctx context.Context, key string) ([]model.Offer, bool, error) {
	return r.database.LoadOffers(ctx, key)
}

func (r *Repository) SaveOffers(ctx context.Context, key string, offers []model.Offer, ttl time.Duration) error {
	return r.database.SaveOffers(ctx, key, offers, ttl)
}

func (r *Repository) DropOffers(ctx context.Context, key string) error {
	return r.database.DropOffers(ctx, key)
}

func (r *Repository) CheckHealth(ctx context.Context) error {
	return r.database.CheckHealth(ctx)
}

func (r *Repository) ListSourceHealth(ctx context.Context) ([]model.SourceHealth, error) {
	return r.database.ListSourceHealth(ctx)
}

func (r *Repository) AwaitSource(ctx context.Context, domain string) error {
	return r.database.AwaitSource(ctx, domain)
}

func (r *Repository) RecordSource(ctx context.Context, domain string, latency time.Duration, sourceErr error) error {
	return r.database.RecordSource(ctx, domain, latency, sourceErr)
}

func (r *Repository) CreateOrder(ctx context.Context, key, hash string, order model.Order) (model.Order, error) {
	return r.database.CreateOrder(ctx, key, hash, order)
}

func (r *Repository) GetOrder(ctx context.Context, id string) (model.Order, error) {
	return r.database.GetOrder(ctx, id)
}

func (r *Repository) MoveOrderStatus(ctx context.Context, id string, from, to model.OrderStatus) (model.Order, error) {
	return r.database.MoveOrderStatus(ctx, id, from, to)
}

func (r *Repository) Close() error { return r.database.CloseStore() }
