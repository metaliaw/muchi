// Package gcp Composes the existing cloud adapters without changing the application flow.
package gcp

import (
	"context"
	"errors"
	"log/slog"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/db"
	"github.com/cangrejometralleta/muchi-api/internal/hosting"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

var (
	_ repositories.Database  = (*db.Store)(nil)
	_ application.Dispatcher = (*taskqueue.Queue)(nil)
)

func BuildLoader(logger *slog.Logger, dispatch bool) *hosting.Loader {
	return &hosting.Loader{Logger: logger, Build: func(ctx context.Context) (hosting.Instance, error) {
		return OpenInstance(ctx, logger, dispatch)
	}}
}

func OpenInstance(ctx context.Context, logger *slog.Logger, dispatch bool) (hosting.Instance, error) {
	settings, err := config.LoadConfig()
	if err != nil {
		return hosting.Instance{}, err
	}
	cloud, err := LoadSettings()
	if err != nil {
		return hosting.Instance{}, err
	}
	database, err := db.OpenStore(ctx, cloud.ProjectID)
	if err != nil {
		return hosting.Instance{}, err
	}
	vault := repositories.New(database, settings.SearchTTL)
	vault.TellStore(logger)
	queue, err := openDispatcher(ctx, cloud, dispatch)
	if err != nil {
		return hosting.Instance{}, errors.Join(err, database.CloseStore())
	}
	close := func() error {
		var queueError error
		if queue != nil {
			queueError = queue.CloseQueue()
		}
		return errors.Join(queueError, database.CloseStore())
	}
	dependencies := application.Dependencies{Vault: vault}
	if queue != nil {
		dependencies.Queue = queue
	}
	runtime, err := application.BuildRuntime(settings, logger, dependencies)
	if err != nil {
		return hosting.Instance{}, errors.Join(err, close())
	}
	return hosting.Instance{
		Operations: application.BuildOperations(runtime, settings, logger),
		Settings:   settings, Close: close,
	}, nil
}

func openDispatcher(ctx context.Context, cloud Settings, dispatch bool) (*taskqueue.Queue, error) {
	if !dispatch || cloud.TaskURL == "" {
		return nil, nil
	}
	return taskqueue.OpenQueue(ctx, cloud.ProjectID, cloud.TaskRegion, cloud.TaskQueue,
		cloud.TaskURL, cloud.TaskAccount, cloud.TaskDeadline)
}
