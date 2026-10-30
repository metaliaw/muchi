package application

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/catalog"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/constants"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/orders"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores/commerce"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/sweep"
)

// Vault Names every Role the one persistence Object Plays for a Runtime: it
// Keeps the Searches and their Items, Caches the Offers, Places and Moves the
// Orders, Answers for its own Health, Counts the Work that Waits and Paces the
// Traffic toward each Source.
//
// The Roles are Listed apart because they are Independent: the Composition Root
// Joins them in one Object, while the Runtime Exposes only Ports.
type Vault interface {
	search.SearchRepository
	search.OfferCache
	moxfield.InventoryCache
	search.OfferReader
	search.HealthStore
	search.OrderRepository
	sweep.WaitingCounter
	source.TrafficGate
	orders.Expirer
}

// Dispatcher Carries Work to a Worker that Runs somewhere else, and Adds a Turn
// when a silent one Spent it.
type Dispatcher interface {
	search.TaskQueue
	sweep.Waker
}

type Runtime struct {
	Service        search.Service
	Vault          Vault
	Queue          Dispatcher
	SupportedGames []directory.GameSupport
	Stores         []directory.ListedStore
	Inventories    moxfield.Shelf
}

// Dependencies Supplies the Ports every host Composes before entering the application.
type Dependencies struct {
	Vault Vault
	Queue Dispatcher
}

// BuildRuntime Composes the same application with the host's chosen Ports.
func BuildRuntime(config config.Config, logger *slog.Logger, dependencies Dependencies) (Runtime, error) {
	vault := dependencies.Vault
	if vault == nil {
		return Runtime{}, errors.New("runtime needs persistence")
	}
	storeConfig, err := directory.LoadStoreConfig(config.StoresPath, logger)
	if err != nil {
		return Runtime{}, err
	}
	fetcher := buildSourceClient(config, vault, logger)
	shops := commerce.Shops{
		Config: storeConfig, Fetcher: fetcher, Sessions: fetcher,
		BuyerName: config.OrderBuyerName, BuyerEmail: config.OrderBuyerEmail,
	}
	supportedGames := directory.SupportedGames(storeConfig)
	listedStores := directory.ListedStores(storeConfig)
	builder := catalog.Builder{Shops: shops, Cache: vault, TTL: config.InventoryCacheTTL, Logger: logger}
	sourcesByGame := builder.SourcesByGame()
	inventories := catalog.CollectInventories(sourcesByGame)
	// The Shelf Checks its own Stock: a List has no Product Page to Visit.
	shops.Lists = inventories
	service := search.Service{
		SearchService: search.SearchService{
			MaxCards:    config.MaxCardsPerSearch,
			MaxQuantity: config.MaxQuantityPerCard,
		},
		CardService: search.CardService{
			Metadata:     builder.MetadataProviders(),
			Autocomplete: builder.AutocompleteProviders(),
			Names:        builder.NameProviders(),
			Languages:    gameLanguages(directory.SupportedLanguages(storeConfig)),
		},
		OfferService: search.OfferService{
			Providers:         builder.Providers(),
			PrintsByGame:      builder.PrintLibraries(),
			SetsByGame:        builder.SetLibraries(),
			SourcesByGame:     sourcesByGame,
			Cache:             vault,
			Offers:            vault,
			CacheNamespace:    search.HashPayload([]any{"search-providers-v9", storeConfig}) + ":",
			CacheTTL:          config.OfferCacheTTL,
			EmptyCacheTTL:     config.OfferCacheEmptyTTL,
			SuspiciousPercent: config.SuspiciousPricePercent,
			StoreLocations:    builder.StoreLocations(),
			SinglesOnly:       builder.SinglesOnly(),
			SealedOnly:        builder.SealedOnly(),
		},
		StockService:    search.StockService{Stocks: shops},
		CheckoutService: search.CheckoutService{Checkouts: shops, Quotes: shops},
		OrderService:    search.OrderService{Orders: vault, Orderer: shops, Checkouts: shops},
	}.WithRepository(vault)
	service.Tasks = dependencies.Queue
	return Runtime{
		Service: service, Vault: vault, Queue: dependencies.Queue,
		SupportedGames: supportedGames, Stores: listedStores, Inventories: inventories,
	}, nil
}

// gameLanguages Types the Keys the Directory Reads as Text.
func gameLanguages(declared map[string][]string) map[model.Game][]string {
	languages := make(map[model.Game][]string, len(declared))
	for key, codes := range declared {
		languages[model.Game(key)] = codes
	}
	return languages
}

func buildSourceClient(config config.Config, gate source.TrafficGate, logger *slog.Logger) source.Client {
	return source.Client{
		HTTP:         &http.Client{Timeout: config.HTTPTimeout},
		Gate:         gate,
		Logger:       logger,
		UserAgent:    constants.UserAgent,
		MaxAttempts:  config.SourceMaxAttempts,
		BaseDelay:    config.SourceRetryBaseDelay,
		MaxBodyBytes: config.SourceMaxBodyBytes,
	}
}
