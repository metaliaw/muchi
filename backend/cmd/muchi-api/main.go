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

	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/httpapi"
	"github.com/cangrejometralleta/muchi-api/internal/metrics"
	"github.com/cangrejometralleta/muchi-api/internal/postgres"
	"github.com/cangrejometralleta/muchi-api/internal/scryfall"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
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
	service, health, err := buildService(config, logger)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if args[0] == "serve" {
		return serveAPI(ctx, config, logger, service, health)
	}
	return workSearches(ctx, config, service)
}

func buildService(config config.Config, logger *slog.Logger) (search.Service, search.HealthStore, error) {
	store, err := postgres.OpenStore(config.DatabaseURL)
	if err != nil {
		return search.Service{}, nil, err
	}
	storeConfig, err := stores.LoadStoreConfig(config.StoresPath)
	if err != nil {
		return search.Service{}, nil, err
	}
	fetcher := source.Client{HTTP: &http.Client{Timeout: config.HTTPTimeout}, Gate: store, Logger: logger, UserAgent: "muchi-api/1.0", MaxAttempts: 3, BaseDelay: 250 * time.Millisecond}
	catalog := scryfall.Client{Fetcher: fetcher, BaseURL: config.ScryfallURL}
	checker := stores.Checker{Fetcher: fetcher, Config: storeConfig}
	service := search.Service{Searches: store, Sources: []search.OfferSource{catalog}, Stocks: checker, Cache: store}
	return service, store, nil
}

func serveAPI(ctx context.Context, config config.Config, logger *slog.Logger, service search.Service, health search.HealthStore) error {
	api := httpapi.API{Searches: service, Health: health, Token: config.APIToken, Logger: logger}
	observed := metrics.NewMetrics().MeasureRequests(api.Handler())
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", observed)
	server := &http.Server{Addr: config.Address, Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go stopServer(ctx, server)
	logger.Info("API Started", "address", config.Address)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func workSearches(ctx context.Context, config config.Config, service search.Service) error {
	worker := search.Worker{Store: service.Searches, Service: service, Owner: config.WorkerID, LeaseDuration: config.LeaseDuration, PollInterval: config.PollInterval}
	return worker.RunWorker(ctx)
}

func stopServer(ctx context.Context, server *http.Server) {
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
}
