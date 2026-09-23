package application

import (
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/aggregators/scrycl"
	"github.com/cangrejometralleta/muchi-api/internal/aggregators/tcgmatch"
	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"github.com/cangrejometralleta/muchi-api/internal/stores/jumpseller"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
	"github.com/cangrejometralleta/muchi-api/internal/stores/prestashop"
	"github.com/cangrejometralleta/muchi-api/internal/stores/shopify"
	"github.com/cangrejometralleta/muchi-api/internal/stores/woocommerce"
)

const setCatalogTTL = 24 * time.Hour

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
			if !store.Enabled || !store.IsSearched() || !slices.Contains(store.Games, key) {
				continue
			}
			result[search.Game(key)] = appendConfiguredSources(result[search.Game(key)], fetcher, domain, store, allowed, cache, ttl, logger)
		}
	}
	return result
}

func appendConfiguredSources(sources []search.OfferSource, fetcher stores.SourceFetcher, domain string, store stores.StoreConfig, allowed map[string]bool, cache moxfield.InventoryCache, ttl time.Duration, logger *slog.Logger) []search.OfferSource {
	if allowed["moxfield"] {
		for _, list := range store.Lists {
			sources = append(sources, &moxfield.Client{Fetcher: fetcher, Cache: cache, TTL: ttl, Logger: logger, StoreID: domain, Store: store.Name, Label: list.Label, ListURL: list.URL, Rate: list.CLPPerCKUSD})
		}
	}
	switch store.Platform {
	case "jumpseller":
		if allowed["jumpseller"] {
			sources = append(sources, jumpseller.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
	case "shopify":
		if allowed["shopify"] {
			sources = append(sources, shopify.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
	case "woocommerce":
		if allowed["woocommerce"] {
			sources = append(sources, woocommerce.Client{Fetcher: fetcher, Domain: domain, Name: store.Name})
		}
	case "prestashop":
		if allowed["prestashop"] {
			sources = append(sources, prestashop.Client{Fetcher: fetcher, Domain: domain, Name: store.Name, SearchPath: "/busqueda"})
		}
	}
	return sources
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

// buildSetLibraries Lends every Enabled Game its Sets from TCGMatch.
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
		libraries[search.Game(name)] = &tcgmatch.SetCatalog{Fetcher: fetcher, BaseURL: provider.URL, Game: name, TTL: setCatalogTTL}
	}
	return libraries
}

// buildSinglesOnly and buildSealedOnly Name Sources by the Product Kind They Sell.
func buildSealedOnly(config stores.Config) map[string]bool {
	named := map[string]bool{}
	for domain, store := range config.Stores {
		if !store.SellsSingles() {
			named[domain] = true
		}
	}
	return named
}

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
			case "scrycl":
				providers[search.Game(game)] = scrycl.Client{Fetcher: fetcher, BaseURL: provider.URL, ExcludeCommunity: !provider.Community}
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

// buildOfferSources Combines Scry.cl with Enabled Store Catalogs. A nil Catalog Leaves Scry.cl Out.
func buildOfferSources(fetcher stores.SourceFetcher, catalog search.OfferSource, config stores.Config, cache moxfield.InventoryCache, ttl time.Duration, logger *slog.Logger) []search.OfferSource {
	sources := []search.OfferSource{}
	if catalog != nil {
		sources = append(sources, catalog)
	}
	for domain, store := range config.Stores {
		if !store.Enabled || !store.IsSearched() {
			continue
		}
		sources = appendConfiguredSources(sources, fetcher, domain, store, allowedOrigins(store.Platform, store.Lists), cache, ttl, logger)
	}
	return sources
}

func allowedOrigins(platform string, lists []stores.MoxfieldList) map[string]bool {
	allowed := map[string]bool{platform: true}
	if len(lists) > 0 {
		allowed["moxfield"] = true
	}
	return allowed
}
