package catalog

import (
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/aggregators/tcgmatch"
	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

const setCatalogTTL = 24 * time.Hour

// PrintLibraries Lends Printings to the Games that Have a Catalog of them.
// Scryfall Knows Magic; the other Games Keep the Image their Source Sends.
func (b Builder) PrintLibraries() map[model.Game]search.PrintLibrary {
	libraries := make(map[model.Game]search.PrintLibrary)
	if game, found := b.Shops.Config.Games[string(model.GameMagic)]; found && game.Enabled {
		libraries[model.GameMagic] = cardmetadata.Scryfall{Fetcher: b.Shops.Fetcher}
	}
	return libraries
}

// SetLibraries Lends every Enabled Game its Sets from TCGMatch.
func (b Builder) SetLibraries() map[model.Game]search.SetLibrary {
	provider, found := b.Shops.Config.SearchProviders["tcgmatch"]
	if !found || !provider.Enabled {
		return nil
	}
	libraries := make(map[model.Game]search.SetLibrary, len(b.Shops.Config.Games))
	for name, game := range b.Shops.Config.Games {
		if !game.Enabled {
			continue
		}
		libraries[model.Game(name)] = &tcgmatch.SetCatalog{Fetcher: b.Shops.Fetcher, BaseURL: provider.URL, Game: name, TTL: setCatalogTTL}
	}
	return libraries
}
