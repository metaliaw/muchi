package directory

import (
	"slices"
	"sort"
)

// GameSupport Says what a Game can be Asked for: Cards, Boxes, or both.
//
// Nobody Declares this per Game. Each Source Declares what it Sells, and the
// Game Inherits the Sum: a Game Supports Cards when some Source that Answers
// for it Sells Cards, and Boxes when some Source Sells Boxes. Mitos y Leyendas
// Reaches both Halves from two different Sources, and neither one Knows it.
type GameSupport struct {
	Key     string `json:"reference_key"`
	Name    string `json:"name"`
	Singles bool   `json:"singles"`
	Sealed  bool   `json:"sealed"`
	// Languages Lists the Languages the Game's Card Names Read in, in the
	// Order the Config Declares them. A Game that Declares none Omits the Field.
	Languages []Language `json:"languages,omitempty"`
}

// Language Is a Code with what a Box needs to Draw it.
type Language struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Example string `json:"example"`
}

// Asked Answers whether this Game can be Asked for that Kind at all.
func (g GameSupport) Asked(sealed bool) bool {
	if sealed {
		return g.Sealed
	}
	return g.Singles
}

// SupportedGames Reduces every Enabled Game to what its Sources Sell.
//
// A Game whose Sources Sell nothing Stays out: Offering it would Promise a
// Search that Answers empty however it is Asked.
func SupportedGames(config Config) []GameSupport {
	supported := make([]GameSupport, 0, len(config.Games))
	for key, game := range config.Games {
		if !game.Enabled {
			continue
		}
		found := GameSupport{Key: key, Name: game.Name, Languages: languagesOf(config, game)}
		for _, provider := range config.SearchProviders {
			if !provider.Enabled || !slices.Contains(provider.Games, key) {
				continue
			}
			found.Singles = found.Singles || provider.ServesSingles()
			found.Sealed = found.Sealed || provider.ServesSealed()
		}
		for _, store := range config.Stores {
			if !storeAnswers(store, game, key) {
				continue
			}
			found.Singles = found.Singles || store.SellsSingles()
			found.Sealed = found.Sealed || store.SellsSealed()
		}
		if found.Singles || found.Sealed {
			supported = append(supported, found)
		}
	}
	// El Orden es del Nombre, no del Mapa: una Lista que Cambia de Orden entre
	// dos Llamadas Mueve el Selector debajo de quien lo Está mirando.
	sort.Slice(supported, func(one, other int) bool {
		return supported[one].Name < supported[other].Name
	})
	return supported
}

// storeAnswers Repeats the Rule that Builds the Sources, so what the Selector
// Offers and what the Search Asks cannot Drift apart.
func storeAnswers(store StoreConfig, game GameConfig, key string) bool {
	if !store.Enabled || !store.IsSearched() || !slices.Contains(store.Games, key) {
		return false
	}
	if len(store.Lists) > 0 && slices.Contains(game.Origins, "moxfield") {
		return true
	}
	return slices.Contains(game.Origins, store.Platform)
}

// languagesOf Reads a Game's Declared Languages against the Catalog. Validation
// Already Refused a Code the Catalog does not Know.
func languagesOf(config Config, game GameConfig) []Language {
	var languages []Language
	for _, code := range game.Languages.Supported {
		entry := config.Languages[code]
		languages = append(languages, Language{Code: code, Label: entry.Label, Example: entry.Example})
	}
	return languages
}

// SupportedLanguages Maps each Game that Declares Languages to their Codes.
// A Game Missing from the Map Declared none, and a Caller may Name any.
func SupportedLanguages(config Config) map[string][]string {
	supported := map[string][]string{}
	for key, game := range config.Games {
		if game.Enabled && game.Languages.Declared() {
			supported[key] = append([]string{}, game.Languages.Supported...)
		}
	}
	return supported
}
