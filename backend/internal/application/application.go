package application

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	firestorestore "github.com/cangrejometralleta/muchi-api/internal/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/prestashop"
	"github.com/cangrejometralleta/muchi-api/internal/scry"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
	"github.com/cangrejometralleta/muchi-api/internal/tcgmatch"
)

// userAgent identifies this service to card sources; it does not vary by environment.
const userAgent = "muchi-api/1.0"

type Runtime struct {
	Service               search.Service
	Store                 *firestorestore.Store
	Queue                 *taskqueue.Queue
	CardMetadataProviders map[search.Game]cardmetadata.Provider
	AutocompleteProviders map[search.Game]cardmetadata.AutocompleteProvider
	SupportedGames        map[search.Game]string
}

// BuildRuntime Casts the Search Providers for one Function Instance.
func BuildRuntime(ctx context.Context, config config.Config, logger *slog.Logger, dispatch bool) (Runtime, error) {
	store, err := firestorestore.OpenStore(ctx, config.ProjectID, config.SearchTTL)
	if err != nil {
		return Runtime{}, err
	}
	// The Store Warns when a Claim steps over dead Items; silent, that Warning
	// is the one that took two Hours to notice.
	store.TellStore(logger)
	storeConfig, err := stores.LoadStoreConfig(config.StoresPath, logger)
	if err != nil {
		return Runtime{}, err
	}
	fetcher := buildSourceClient(config, store, logger)
	tcgmatchMetadata := tcgmatch.Client{Fetcher: fetcher, BaseURL: storeConfig.SearchProviders["tcgmatch"].URL, Game: string(search.GamePokemon)}
	tcgmatchYuGiOh := tcgmatch.Client{Fetcher: fetcher, BaseURL: storeConfig.SearchProviders["tcgmatch"].URL, Game: string(search.GameYuGiOh)}
	cardMetadataProviders := map[search.Game]cardmetadata.Provider{
		search.GameMagic:   cardmetadata.Scryfall{Fetcher: fetcher},
		search.GamePokemon: tcgmatchMetadata,
		search.GameYuGiOh:  tcgmatchYuGiOh,
	}
	autocompleteProviders := map[search.Game]cardmetadata.AutocompleteProvider{
		search.GameMagic:   cardmetadata.Scryfall{Fetcher: fetcher},
		search.GamePokemon: tcgmatchMetadata,
		search.GameYuGiOh:  tcgmatchYuGiOh,
	}
	supportedGames := make(map[search.Game]string)
	for key, game := range storeConfig.Games {
		if game.Enabled {
			supportedGames[search.Game(key)] = game.Name
		}
	}
	checker := stores.Checker{Fetcher: fetcher, Config: storeConfig}
	service := search.Service{
		Searches:          store,
		Providers:         buildProviders(fetcher, storeConfig),
		PrintsByGame:      buildPrintLibraries(fetcher, storeConfig),
		SourcesByGame:     buildSourcesByGame(fetcher, storeConfig, store, config.OfferCacheTTL, logger),
		Stocks:            checker,
		Cache:             store,
		CacheNamespace:    search.HashPayload([]any{"search-providers-v6", storeConfig}) + ":",
		CacheTTL:          config.OfferCacheTTL,
		EmptyCacheTTL:     config.OfferCacheEmptyTTL,
		MaxCards:          config.MaxCardsPerSearch,
		MaxQuantity:       config.MaxQuantityPerCard,
		SuspiciousPercent: config.SuspiciousPricePercent,
	}
	if dispatch && config.TaskURL != "" {
		queue, err := taskqueue.OpenQueue(
			ctx, config.ProjectID, config.TaskRegion, config.TaskQueue,
			config.TaskURL, config.TaskAccount, config.TaskDeadline,
		)
		if err != nil {
			return Runtime{}, err
		}
		service.Tasks = queue
		return Runtime{Service: service, Store: store, Queue: queue, CardMetadataProviders: cardMetadataProviders, AutocompleteProviders: autocompleteProviders, SupportedGames: supportedGames}, nil
	}
	return Runtime{Service: service, Store: store, CardMetadataProviders: cardMetadataProviders, AutocompleteProviders: autocompleteProviders, SupportedGames: supportedGames}, nil
}

func buildSourcesByGame(fetcher stores.SourceFetcher, config stores.Config, cache moxfield.InventoryCache, ttl time.Duration, logger *slog.Logger) map[search.Game][]search.OfferSource {
	result := make(map[search.Game][]search.OfferSource)
	for key, game := range config.Games {
		if !game.Enabled {
			continue
		}
		allowed := make(map[string]bool, len(game.Origins))
		for _, origin := range game.Origins {
			allowed[origin] = true
		}
		for domain, store := range config.Stores {
			if !store.Enabled || !slices.Contains(store.Games, key) {
				continue
			}
			for _, list := range store.Lists {
				if allowed["moxfield"] {
					result[search.Game(key)] = append(result[search.Game(key)], &moxfield.Client{Fetcher: fetcher, Cache: cache, TTL: ttl, Logger: logger, StoreID: domain, Store: store.Name, Label: list.Label, ListURL: list.URL, Rate: list.CLPPerCKUSD})
				}
			}
			switch store.Platform {
			case "jumpseller":
				if allowed["jumpseller"] {
					result[search.Game(key)] = append(result[search.Game(key)], jumpseller.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
				}
			case "shopify":
				if allowed["shopify"] {
					result[search.Game(key)] = append(result[search.Game(key)], shopify.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
				}
			case "woocommerce":
				if allowed["woocommerce"] {
					result[search.Game(key)] = append(result[search.Game(key)], stores.Catalog{Fetcher: fetcher, Domain: domain, Name: store.Name})
				}
			case "prestashop":
				if allowed["prestashop"] {
					result[search.Game(key)] = append(result[search.Game(key)], prestashop.Client{Fetcher: fetcher, Domain: domain, Name: store.Name, SearchPath: "/busqueda"})
				}
			}
		}
	}
	return result
}

// buildPrintLibraries Lends Printings to the Games that Have a Catalog of them.
// Scryfall Knows Magic; the other Games Keep the Image their Source Sends.
func buildPrintLibraries(fetcher stores.SourceFetcher, config stores.Config) map[search.Game]search.PrintLibrary {
	libraries := make(map[search.Game]search.PrintLibrary)
	if game, found := config.Games[string(search.GameMagic)]; found && game.Enabled {
		libraries[search.GameMagic] = cardmetadata.Scryfall{Fetcher: fetcher}
	}
	return libraries
}

func buildProviders(fetcher stores.SourceFetcher, config stores.Config) map[search.Game]search.Provider {
	providers := make(map[search.Game]search.Provider, len(config.Games))
	for _, provider := range config.SearchProviders {
		if !provider.Enabled {
			continue
		}
		for _, game := range provider.Games {
			if gameConfig, found := config.Games[game]; !found || !gameConfig.Enabled {
				continue
			}
			switch provider.Type {
			case "scry":
				providers[search.Game(game)] = scry.Client{Fetcher: fetcher, BaseURL: provider.URL, ExcludeCommunity: !provider.Community}
			case "tcgmatch":
				providers[search.Game(game)] = tcgmatch.Client{Fetcher: fetcher, BaseURL: provider.URL, Game: game}
			}
		}
	}
	return providers
}

func buildSourceClient(config config.Config, gate source.TrafficGate, logger *slog.Logger) source.Client {
	return source.Client{
		HTTP:         &http.Client{Timeout: config.HTTPTimeout},
		Gate:         gate,
		Logger:       logger,
		UserAgent:    userAgent,
		MaxAttempts:  config.SourceMaxAttempts,
		BaseDelay:    config.SourceRetryBaseDelay,
		MaxBodyBytes: config.SourceMaxBodyBytes,
	}
}

// buildOfferSources Combines Scry with Enabled Store Catalogs. A nil Catalog Leaves Scry Out.
func buildOfferSources(fetcher stores.SourceFetcher, catalog search.OfferSource, config stores.Config, cache moxfield.InventoryCache, ttl time.Duration, logger *slog.Logger) []search.OfferSource {
	sources := []search.OfferSource{}
	if catalog != nil {
		sources = append(sources, catalog)
	}
	for domain, store := range config.Stores {
		if !store.Enabled {
			continue
		}
		for _, list := range store.Lists {
			sources = append(sources, &moxfield.Client{Fetcher: fetcher, Cache: cache, TTL: ttl, Logger: logger, StoreID: domain, Store: store.Name, Label: list.Label, ListURL: list.URL, Rate: list.CLPPerCKUSD})
		}
		if store.Platform == "jumpseller" {
			sources = append(sources, jumpseller.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
		if store.Platform == "shopify" {
			sources = append(sources, shopify.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
		if store.Platform == "woocommerce" {
			sources = append(sources, stores.Catalog{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
		if store.Platform == "prestashop" {
			sources = append(sources, prestashop.Client{Fetcher: fetcher, Domain: domain, Name: store.Name, SearchPath: "/busqueda"})
		}
	}
	return sources
}
