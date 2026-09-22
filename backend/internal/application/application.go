package application

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
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
	SupportedGames        map[search.Game]string
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

func buildStoreLocations(config stores.Config) map[string][]string {
	result := make(map[string][]string)
	for domain, store := range config.Stores {
		locations := formatStoreLocations(store.Locations)
		if len(locations) == 0 {
			continue
		}
		result[domain] = locations
		if store.Name != "" {
			result[store.Name] = locations
		}
	}
	return result
}

func formatStoreLocations(locations []stores.StoreLocation) []string {
	result := make([]string, 0, len(locations))
	for _, location := range locations {
		detail := location.District
		if detail == "" {
			detail = location.Pickup
		}
		result = append(result, strings.Join(removeEmpty([]string{location.City, detail}), " - "))
	}
	return result
}

func removeEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
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

// collectInventories Shelves each Moxfield List once, however many Games Play it.
func collectInventories(sourcesByGame map[search.Game][]search.OfferSource) moxfield.Shelf {
	shelf := moxfield.Shelf{}
	seen := make(map[*moxfield.Client]bool)
	for _, sources := range sourcesByGame {
		for _, source := range sources {
			list, ok := source.(*moxfield.Client)
			if !ok || seen[list] {
				continue
			}
			seen[list] = true
			shelf.Lists = append(shelf.Lists, list)
		}
	}
	return shelf
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

// setCatalogTTL Holds a Set List for a Day. A Set Releases a few Times a Year.
const setCatalogTTL = 24 * time.Hour

// buildSetLibraries Lends every Enabled Game its Sets. TCGMatch Lists the Sets
// of the four Games it Knows, Magic among them, even where it is not the
// Provider that Answers that Game.
func buildSetLibraries(fetcher stores.SourceFetcher, config stores.Config) map[search.Game]search.SetLibrary {
	provider, found := config.SearchProviders["tcgmatch"]
	if !found || !provider.Enabled {
		return nil
	}
	libraries := make(map[search.Game]search.SetLibrary, len(config.Games))
	for name, game := range config.Games {
		if !game.Enabled {
			continue
		}
		libraries[search.Game(name)] = &tcgmatch.SetCatalog{
			Fetcher: fetcher, BaseURL: provider.URL, Game: name, TTL: setCatalogTTL,
		}
	}
	return libraries
}

// buildSinglesOnly Names the Sources their Configuration Marks `sealed: false`,
// the Way each one Names itself in a Failure. An Aggregator Answers by the Host
// of its URL; a Store, by its Domain.
func buildSinglesOnly(config stores.Config) map[string]bool {
	named := map[string]bool{}
	for _, provider := range config.SearchProviders {
		if provider.ServesSealed() {
			continue
		}
		if host := readHost(provider.URL); host != "" {
			named[host] = true
		}
	}
	for domain, store := range config.Stores {
		if !store.SellsSealed() {
			named[domain] = true
		}
	}
	return named
}

// readHost Names a Provider the Way its Client Does, so the Mark and the Fault
// Speak of the same Source.
func readHost(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}
	return parsed.Host
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
				providers[search.Game(game)] = buildTCGMatch(fetcher, config, game)
			}
		}
	}
	return providers
}

func buildTCGMatch(fetcher stores.SourceFetcher, config stores.Config, game string) tcgmatch.Client {
	client := tcgmatch.Client{Fetcher: fetcher, BaseURL: config.SearchProviders["tcgmatch"].URL, Game: game}
	if game == string(search.GamePokemon) {
		client.PokemonCatalogURL = config.SearchProviders["tcgdex"].URL
	}
	return client
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
