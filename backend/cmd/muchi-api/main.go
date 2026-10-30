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
	"github.com/cangrejometralleta/muchi-api/internal/gcp"
	"github.com/cangrejometralleta/muchi-api/internal/hosting"
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

func runCommand(logger *slog.Logger, args []string) (result error) {
	if len(args) != 1 || !validCommand(args[0]) {
		return errors.New("usage: muchi-api serve|work|process|sweep|release")
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	loader := gcp.BuildLoader(logger, args[0] == "serve" || args[0] == "sweep")
	instance, err := loader.LoadInstance()
	if err != nil {
		return err
	}
	defer func() {
		if result == context.Canceled {
			result = nil
		}
		result = errors.Join(result, loader.CloseInstance())
	}()
	return executeCommand(ctx, logger, args[0], instance)
}

func validCommand(command string) bool {
	switch command {
	case "serve", "work", "process", "sweep", "release":
		return true
	default:
		return false
	}
}

func executeCommand(ctx context.Context, logger *slog.Logger, command string, instance hosting.Instance) error {
	switch command {
	case "serve":
		return serveAPI(ctx, instance.Settings, logger, instance)
	case "work":
		return instance.Operations.WorkSearches(ctx)
	case "process":
		err := instance.Operations.ProcessSearch(ctx, instance.Settings.WorkerID)
		if errors.Is(err, search.ErrNotFound) {
			return nil
		}
		return err
	case "sweep":
		count, err := instance.Operations.SweepQueue(ctx)
		if err == nil {
			logger.Info("Queue Swept", "woken", count)
		}
		return err
	case "release":
		count, err := instance.Operations.ReleaseOrders(ctx)
		if err == nil {
			logger.Info("Orders Released", "released", count)
		}
		return err
	default:
		return errors.New("unknown command")
	}
}

func serveAPI(ctx context.Context, config config.Config, logger *slog.Logger, instance hosting.Instance) error {
	observed := metrics.NewMetrics().MeasureRequests(instance.Operations.API)
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.BuildHandler())
	mux.Handle("/", observed)
	server := buildAPIServer(config, mux)
	lifecycle, cancel := context.WithCancel(ctx)
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- stopServer(lifecycle, server, config.ServerShutdownTimeout) }()
	logger.Info("API Started", "address", config.Address)
	err := server.ListenAndServe()
	cancel()
	shutdownError := <-stopped
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	return errors.Join(err, shutdownError)
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

func stopServer(ctx context.Context, server *http.Server, timeout time.Duration) error {
	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		return errors.Join(err, server.Close())
	}
	return nil
}
