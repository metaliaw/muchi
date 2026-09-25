package application

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/catalog"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/constants"
	firestorestore "github.com/cangrejometralleta/muchi-api/internal/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/sweep"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

// Vault Names every Role one Store Plays for a Runtime: it Keeps the Searches,
// Caches the Offers, Answers for its own Health, Counts the Work that Waits and
// Paces the Traffic toward each Source.
//
// The Roles are Listed apart because they are Independent: the Composition Root
// Happens to Satisfy all five with one Firestore Store, and nothing above it
// Depends on that. Only this Package Names a Vendor.
type Vault interface {
	search.SearchRepository
	search.SearchItemRepository
	search.OfferCache
	search.HealthStore
	sweep.WaitingCounter
	source.TrafficGate
}

// Dispatcher Carries Work to a Worker that Runs somewhere else, and Adds a Turn
// when a silent one Spent it.
type Dispatcher interface {
	search.TaskQueue
	sweep.Waker
}

// Teller Takes a Logger from a Store that has something to Say. A Store that
// Stays quiet simply does not Implement it.
type Teller interface {
	TellStore(*slog.Logger)
}

type Runtime struct {
	Service               search.Service
	Store                 Vault
	Queue                 Dispatcher
	CardMetadataProviders map[search.Game]cardmetadata.Provider
	AutocompleteProviders map[search.Game]cardmetadata.AutocompleteProvider
	SupportedGames        []stores.GameSupport
	Inventories           moxfield.Shelf
}

// BuildRuntime Casts the Search Providers for one Function Instance.
func BuildRuntime(ctx context.Context, config config.Config, logger *slog.Logger, dispatch bool) (Runtime, error) {
	store, err := firestorestore.OpenStore(ctx, config.ProjectID, config.SearchTTL)
	if err != nil {
		return Runtime{}, err
	}
	// The Store Warns when a Claim steps over dead Items; silent, that Warning
	// is the one that took two Hours to notice.
	if teller, told := any(store).(Teller); told {
		teller.TellStore(logger)
	}
	storeConfig, err := stores.LoadStoreConfig(config.StoresPath, logger)
	if err != nil {
		return Runtime{}, err
	}
	fetcher := buildSourceClient(config, store, logger)
	tcgmatchMetadata := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GamePokemon))
	tcgmatchYuGiOh := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GameYuGiOh))
	tcgmatchOnePiece := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GameOnePiece))
	tcgmatchDigimon := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GameDigimon))
	tcgmatchRiftbound := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GameRiftbound))
	tcgmatchMitos := catalog.BuildTCGMatch(fetcher, storeConfig, string(search.GameMitos))
	cardMetadataProviders := map[search.Game]cardmetadata.Provider{
		search.GameMagic:     cardmetadata.Scryfall{Fetcher: fetcher},
		search.GamePokemon:   tcgmatchMetadata,
		search.GameYuGiOh:    tcgmatchYuGiOh,
		search.GameOnePiece:  tcgmatchOnePiece,
		search.GameDigimon:   tcgmatchDigimon,
		search.GameRiftbound: tcgmatchRiftbound,
		search.GameMitos:     tcgmatchMitos,
	}
	autocompleteProviders := map[search.Game]cardmetadata.AutocompleteProvider{
		search.GameMagic:     cardmetadata.Scryfall{Fetcher: fetcher},
		search.GamePokemon:   tcgmatchMetadata,
		search.GameYuGiOh:    tcgmatchYuGiOh,
		search.GameOnePiece:  tcgmatchOnePiece,
		search.GameDigimon:   tcgmatchDigimon,
		search.GameRiftbound: tcgmatchRiftbound,
		search.GameMitos:     tcgmatchMitos,
	}
	supportedGames := stores.SupportedGames(storeConfig)
	sourcesByGame := catalog.BuildSourcesByGame(fetcher, storeConfig, store, config.InventoryCacheTTL, logger)
	inventories := catalog.CollectInventories(sourcesByGame)
	// The Shelf Checks its own Stock: a List has no Product Page to Visit.
	checker := stores.Checker{Fetcher: fetcher, Config: storeConfig, Lists: inventories}
	service := search.Service{
		Repository:        store,
		Providers:         catalog.BuildProviders(fetcher, storeConfig),
		PrintsByGame:      catalog.BuildPrintLibraries(fetcher, storeConfig),
		SetsByGame:        catalog.BuildSetLibraries(fetcher, storeConfig),
		SourcesByGame:     sourcesByGame,
		Stocks:            checker,
		Checkouts:         storeConfig,
		Quotes:            stores.Quoter{Sessions: fetcher, Config: storeConfig},
		Cache:             store,
		CacheNamespace:    search.HashPayload([]any{"search-providers-v9", storeConfig}) + ":",
		CacheTTL:          config.OfferCacheTTL,
		EmptyCacheTTL:     config.OfferCacheEmptyTTL,
		MaxCards:          config.MaxCardsPerSearch,
		MaxQuantity:       config.MaxQuantityPerCard,
		SuspiciousPercent: config.SuspiciousPricePercent,
		StoreLocations:    catalog.BuildStoreLocations(storeConfig),
		SinglesOnly:       catalog.BuildSinglesOnly(storeConfig),
		SealedOnly:        catalog.BuildSealedOnly(storeConfig),
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
		return Runtime{Service: service, Store: store, Queue: queue, CardMetadataProviders: cardMetadataProviders, AutocompleteProviders: autocompleteProviders, SupportedGames: supportedGames, Inventories: inventories}, nil
	}
	return Runtime{Service: service, Store: store, CardMetadataProviders: cardMetadataProviders, AutocompleteProviders: autocompleteProviders, SupportedGames: supportedGames, Inventories: inventories}, nil
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
