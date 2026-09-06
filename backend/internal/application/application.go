package application

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/config"
	firestorestore "github.com/cangrejometralleta/muchi-api/internal/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/scryfall"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

// userAgent identifies this service to card sources; it does not vary by environment.
const userAgent = "muchi-api/1.0"

type Runtime struct {
	Service search.Service
	Store   *firestorestore.Store
}

// BuildRuntime Casts the Search Providers for one Function Instance.
func BuildRuntime(ctx context.Context, config config.Config, logger *slog.Logger, dispatch bool) (Runtime, error) {
	store, err := firestorestore.OpenStore(ctx, config.ProjectID)
	if err != nil {
		return Runtime{}, err
	}
	storeConfig, err := stores.LoadStoreConfig(config.StoresPath)
	if err != nil {
		return Runtime{}, err
	}
	fetcher := buildSourceClient(config, store, logger)
	catalog := scryfall.Client{Fetcher: fetcher, BaseURL: config.ScryfallURL}
	checker := stores.Checker{Fetcher: fetcher, Config: storeConfig}
	service := search.Service{
		Searches:      store,
		Sources:       []search.OfferSource{catalog},
		Stocks:        checker,
		Cache:         store,
		CacheTTL:      config.OfferCacheTTL,
		EmptyCacheTTL: config.OfferCacheEmptyTTL,
	}
	if dispatch && config.TaskURL != "" {
		queue, err := taskqueue.OpenQueue(
			ctx, config.ProjectID, config.TaskRegion, config.TaskQueue,
			config.TaskURL, config.TaskAccount,
		)
		if err != nil {
			return Runtime{}, err
		}
		service.Tasks = queue
	}
	return Runtime{Service: service, Store: store}, nil
}

func buildSourceClient(config config.Config, gate source.TrafficGate, logger *slog.Logger) source.Client {
	return source.Client{
		HTTP:        &http.Client{Timeout: config.HTTPTimeout},
		Gate:        gate,
		Logger:      logger,
		UserAgent:   userAgent,
		MaxAttempts: config.SourceMaxAttempts,
		BaseDelay:   config.SourceRetryBaseDelay,
	}
}
