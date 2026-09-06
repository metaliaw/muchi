package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/httpapi"
	"github.com/cangrejometralleta/muchi-api/internal/metrics"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := runCommand(logger, os.Args[1:]); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("Command Failed", "error", err)
		os.Exit(1)
	}
}

func runCommand(logger *slog.Logger, args []string) error {
	if len(args) != 1 || (args[0] != "serve" && args[0] != "work") {
		return errors.New("usage: muchi-api serve|work")
	}
	config, err := config.LoadConfig()
	if err != nil {
		return err
	}
	runtime, err := application.BuildRuntime(context.Background(), config, logger, args[0] == "serve")
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if args[0] == "serve" {
		return serveAPI(ctx, config, logger, runtime.Service, runtime.Store)
	}
	return workSearches(ctx, config, runtime.Service)
}

func serveAPI(ctx context.Context, config config.Config, logger *slog.Logger, service search.Service, health search.HealthStore) error {
	api := httpapi.API{
		Searches: service, Health: health, Token: config.APIToken, Logger: logger,
		HealthCheckTimeout: config.HealthCheckTimeout,
	}
	observed := metrics.NewMetrics().MeasureRequests(api.BuildHandler())
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.BuildHandler())
	mux.Handle("/", observed)
	server := buildAPIServer(config, mux)
	go stopServer(ctx, server, config.ServerShutdownTimeout)
	logger.Info("API Started", "address", config.Address)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func buildAPIServer(config config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              config.Address,
		Handler:           handler,
		ReadHeaderTimeout: config.ServerReadHeaderTimeout,
		ReadTimeout:       config.ServerReadTimeout,
		WriteTimeout:      config.ServerWriteTimeout,
		IdleTimeout:       config.ServerIdleTimeout,
	}
}

func workSearches(ctx context.Context, config config.Config, service search.Service) error {
	worker := search.Worker{
		Store:           service.Searches,
		Service:         service,
		Owner:           config.WorkerID,
		LeaseDuration:   config.LeaseDuration,
		PollInterval:    config.PollInterval,
		StockCheckLimit: config.StockCheckLimit,
	}
	return worker.RunWorker(ctx)
}

func stopServer(ctx context.Context, server *http.Server, timeout time.Duration) {
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
