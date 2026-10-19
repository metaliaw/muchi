package catalog

import (
	"slices"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
)

// SourcesByGame Groups the Enabled Stores under each Game they Sell.
func (b Builder) SourcesByGame() map[model.Game][]search.OfferSource {
	result := make(map[model.Game][]search.OfferSource)
	for key, game := range b.Shops.Config.Games {
		if !game.Enabled {
			continue
		}
		allowed := make(map[string]bool, len(game.Origins))
		for _, origin := range game.Origins {
			allowed[origin] = true
		}
		for domain, store := range b.Shops.Config.Stores {
			if !store.Enabled || !store.IsSearched() || !slices.Contains(store.Games, key) {
				continue
			}
			result[model.Game(key)] = b.appendConfiguredSources(result[model.Game(key)], domain, store, allowed)
		}
	}
	return result
}

// offerSources Combines Scry.cl with Enabled Store Catalogs. A nil Catalog Leaves Scry.cl Out.
func (b Builder) offerSources(catalog search.OfferSource) []search.OfferSource {
	sources := []search.OfferSource{}
	if catalog != nil {
		sources = append(sources, catalog)
	}
	for domain, store := range b.Shops.Config.Stores {
		if !store.Enabled || !store.IsSearched() {
			continue
		}
		sources = b.appendConfiguredSources(sources, domain, store, allowedOrigins(store.Platform, store.Lists))
	}
	return sources
}

func (b Builder) appendConfiguredSources(sources []search.OfferSource, domain string, store directory.StoreConfig, allowed map[string]bool) []search.OfferSource {
	if allowed["moxfield"] {
		for _, list := range store.Lists {
			sources = append(sources, &moxfield.Client{Fetcher: b.Shops.Fetcher, Cache: b.Cache, TTL: b.TTL, Logger: b.Logger, StoreID: domain, Store: store.Name, Label: list.Label, ListURL: list.URL, Rate: list.CLPPerCKUSD})
		}
	}
	if !allowed[store.Platform] {
		return sources
	}
	if finder, can := b.Shops.Finder(domain); can {
		sources = append(sources, finder)
	}
	return sources
}

func allowedOrigins(platform string, lists []directory.MoxfieldList) map[string]bool {
	allowed := map[string]bool{platform: true}
	if len(lists) > 0 {
		allowed["moxfield"] = true
	}
	return allowed
}

// CollectInventories Shelves each Moxfield List once, however many Games Play it.
func CollectInventories(sourcesByGame map[model.Game][]search.OfferSource) moxfield.Shelf {
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
