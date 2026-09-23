package application

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	firestorestore "github.com/cangrejometralleta/muchi-api/internal/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/taskqueue"
)

// userAgent identifies this service to card sources; it does not vary by environment.
const userAgent = "muchi-api/1.0"

// Vault Names every Role one Store Plays for a Runtime: it Keeps the Searches,
// Caches the Offers, Answers for its own Health, Counts the Work that Waits and
// Paces the Traffic toward each Source.
//
// The Roles are Listed apart because they are Independent: the Composition Root
// Happens to Satisfy all five with one Firestore Store, and nothing above it
// Depends on that. Only this Package Names a Vendor.
type Vault interface {
	search.SearchStore
	search.OfferCache
	search.HealthStore
	search.WaitingCounter
	source.TrafficGate
}

// Dispatcher Carries Work to a Worker that Runs somewhere else, and Adds a Turn
// when a silent one Spent it.
type Dispatcher interface {
	search.TaskQueue
	search.Waker
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
	tcgmatchMetadata := buildTCGMatch(fetcher, storeConfig, string(search.GamePokemon))
	tcgmatchYuGiOh := buildTCGMatch(fetcher, storeConfig, string(search.GameYuGiOh))
	tcgmatchOnePiece := buildTCGMatch(fetcher, storeConfig, string(search.GameOnePiece))
	tcgmatchDigimon := buildTCGMatch(fetcher, storeConfig, string(search.GameDigimon))
	tcgmatchRiftbound := buildTCGMatch(fetcher, storeConfig, string(search.GameRiftbound))
	tcgmatchMitos := buildTCGMatch(fetcher, storeConfig, string(search.GameMitos))
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
	sourcesByGame := buildSourcesByGame(fetcher, storeConfig, store, config.InventoryCacheTTL, logger)
	inventories := collectInventories(sourcesByGame)
	// The Shelf Checks its own Stock: a List has no Product Page to Visit.
	checker := stores.Checker{Fetcher: fetcher, Config: storeConfig, Lists: inventories}
	service := search.Service{
		Searches:          store,
		Providers:         buildProviders(fetcher, storeConfig),
		PrintsByGame:      buildPrintLibraries(fetcher, storeConfig),
		SetsByGame:        buildSetLibraries(fetcher, storeConfig),
		SourcesByGame:     sourcesByGame,
		Stocks:            checker,
		Cache:             store,
		CacheNamespace:    search.HashPayload([]any{"search-providers-v8", storeConfig}) + ":",
		CacheTTL:          config.OfferCacheTTL,
		EmptyCacheTTL:     config.OfferCacheEmptyTTL,
		MaxCards:          config.MaxCardsPerSearch,
		MaxQuantity:       config.MaxQuantityPerCard,
		SuspiciousPercent: config.SuspiciousPricePercent,
		StoreLocations:    buildStoreLocations(storeConfig),
		SinglesOnly:       buildSinglesOnly(storeConfig),
		SealedOnly:        buildSealedOnly(storeConfig),
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
		UserAgent:    userAgent,
		MaxAttempts:  config.SourceMaxAttempts,
		BaseDelay:    config.SourceRetryBaseDelay,
		MaxBodyBytes: config.SourceMaxBodyBytes,
	}
}
