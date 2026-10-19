package catalog

import (
	"github.com/cangrejometralleta/muchi-api/internal/aggregators/scrycl"
	"github.com/cangrejometralleta/muchi-api/internal/aggregators/tcgmatch"
	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// tcgmatchGames Lists the Games whose Card Metadata and Autocomplete TCGMatch
// Answers. Magic Answers through Scryfall instead.
var tcgmatchGames = []model.Game{
	model.GamePokemon, model.GameYuGiOh, model.GameOnePiece,
	model.GameDigimon, model.GameRiftbound, model.GameMitos,
}

// Providers Casts each Enabled Search Provider under the Games it Serves.
func (b Builder) Providers() map[model.Game]search.Provider {
	config := b.Shops.Config
	providers := make(map[model.Game]search.Provider, len(config.Games))
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
				providers[model.Game(game)] = scrycl.Client{Fetcher: b.Shops.Fetcher, BaseURL: provider.URL, ExcludeCommunity: !provider.Community}
			case "tcgmatch":
				providers[model.Game(game)] = b.TCGMatch(game)
			}
		}
	}
	return providers
}

// TCGMatch Casts the TCGMatch Client for one Game.
func (b Builder) TCGMatch(game string) tcgmatch.Client {
	config := b.Shops.Config
	client := tcgmatch.Client{Fetcher: b.Shops.Fetcher, BaseURL: config.SearchProviders["tcgmatch"].URL, Game: game}
	if game == string(model.GamePokemon) {
		client.PokemonCatalogURL = config.SearchProviders["tcgdex"].URL
	}
	return client
}

// scryfallGames Lists the Games whose Config Hands their Card Names to
// Scryfall. A Config that Declares no Language Provider at all keeps Magic
// there, as it Was before the Config Said so.
func (b Builder) scryfallGames() []model.Game {
	games := []model.Game{}
	for key, game := range b.Shops.Config.Games {
		if game.Languages.Provider == "scryfall" {
			games = append(games, model.Game(key))
		}
	}
	if len(games) == 0 {
		return []model.Game{model.GameMagic}
	}
	return games
}

// MetadataProviders Names who Answers a Card's Metadata in each Game.
func (b Builder) MetadataProviders() map[model.Game]cardmetadata.Provider {
	providers := map[model.Game]cardmetadata.Provider{}
	for _, game := range b.scryfallGames() {
		providers[game] = cardmetadata.Scryfall{Fetcher: b.Shops.Fetcher}
	}
	for _, game := range tcgmatchGames {
		providers[game] = b.TCGMatch(string(game))
	}
	return providers
}

// AutocompleteProviders Names who Completes a Card Name in each Game.
func (b Builder) AutocompleteProviders() map[model.Game]cardmetadata.AutocompleteProvider {
	providers := map[model.Game]cardmetadata.AutocompleteProvider{}
	for _, game := range b.scryfallGames() {
		providers[game] = cardmetadata.Scryfall{Fetcher: b.Shops.Fetcher}
	}
	for _, game := range tcgmatchGames {
		providers[game] = b.TCGMatch(string(game))
	}
	return providers
}

// NameProviders Names who Reads a Card Name as the Canonical one in each Game.
// Only the Games the Config Hands to Scryfall Have one: TCGMatch Indexes the
// Names its Shops Write and Knows no other Spelling.
func (b Builder) NameProviders() map[model.Game]cardmetadata.NameProvider {
	providers := map[model.Game]cardmetadata.NameProvider{}
	for _, game := range b.scryfallGames() {
		providers[game] = cardmetadata.Scryfall{Fetcher: b.Shops.Fetcher}
	}
	return providers
}
