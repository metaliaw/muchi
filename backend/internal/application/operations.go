package application

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/handler"
	"github.com/cangrejometralleta/muchi-api/internal/orders"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/sweep"
)

var ErrDispatcherMissing = errors.New("sweep needs a task queue")

// Operations Is the application surface shared by HTTP, task and command hosts.
type Operations struct {
	API      http.Handler
	worker   search.Worker
	sweeper  sweep.Sweeper
	releaser orders.Releaser
}

func BuildOperations(runtime Runtime, settings config.Config, logger *slog.Logger) Operations {
	api := handler.Handler{
		Searches: runtime.Service, Health: runtime.Vault, Token: settings.APIToken,
		Logger: logger, SupportedGames: runtime.SupportedGames, Stores: runtime.Stores,
		Inventories: runtime.Inventories, HealthCheckTimeout: settings.HealthCheckTimeout,
		OrderWebhookSecret: settings.OrderWebhookSecret,
	}
	return Operations{
		API: api.BuildHandler(), worker: BuildSearchWorker(runtime, settings),
		sweeper:  sweep.Sweeper{Items: runtime.Vault, Queue: runtime.Queue, Logger: logger, MaxWakes: settings.SweepMaxWakes},
		releaser: orders.Releaser{Orders: runtime.Vault, Logger: logger, MaxAge: settings.OrderPendingTTL, MaxReleases: settings.OrderReleaseMaxWakes},
	}
}

func (o Operations) ProcessSearch(ctx context.Context, owner string) error {
	worker := o.worker
	if owner != "" {
		worker.Owner = owner
	}
	return worker.ProcessNext(ctx)
}

func (o Operations) WorkSearches(ctx context.Context) error {
	return o.worker.RunWorker(ctx)
}

func (o Operations) SweepQueue(ctx context.Context) (int, error) {
	if o.sweeper.Queue == nil {
		return 0, ErrDispatcherMissing
	}
	return o.sweeper.SweepQueue(ctx)
}

func (o Operations) ReleaseOrders(ctx context.Context) (int, error) {
	return o.releaser.ReleaseOrders(ctx)
}
