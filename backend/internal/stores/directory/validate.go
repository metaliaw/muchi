package directory

import (
	"errors"
	"fmt"
	"net/url"
	"slices"

	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
)

func ValidateConfig(config Config) error {
	if err := validateGames(config.Games); err != nil {
		return err
	}
	if err := validateLanguages(config.Languages, config.Games); err != nil {
		return err
	}
	if err := validateSearchProviders(config.SearchProviders, config.Games, config.Stores); err != nil {
		return err
	}
	for domain, store := range config.Stores {
		if domain == "" || len(store.Games) == 0 {
			return errors.New("invalid store configuration")
		}
		for _, game := range store.Games {
			if _, known := config.Games[game]; !known {
				return fmt.Errorf("unknown game %q for store %q", game, domain)
			}
		}
		if err := validateChecker(store); err != nil {
			return err
		}
		if err := validateLists(store.Lists); err != nil {
			return err
		}
		if err := validateLocations(store.Locations); err != nil {
			return err
		}
	}
	return nil
}

func validateLocations(locations []StoreLocation) error {
	for _, location := range locations {
		if location.City == "" || location.District != "" && location.Pickup != "" {
			return errors.New("invalid store location")
		}
		if (location.Latitude == nil) != (location.Longitude == nil) {
			return errors.New("invalid store location")
		}
		if location.Latitude != nil && (*location.Latitude < -90 || *location.Latitude > 90 || *location.Longitude < -180 || *location.Longitude > 180) {
			return errors.New("invalid store location")
		}
	}
	return nil
}

func validateGames(games map[string]GameConfig) error {
	if len(games) == 0 {
		return errors.New("invalid game configuration")
	}
	for game, config := range games {
		if game == "" || config.Name == "" || config.ReferenceKey != game || len(config.Origins) == 0 {
			return errors.New("invalid game configuration")
		}
		for _, origin := range config.Origins {
			if origin == "" {
				return errors.New("invalid game configuration")
			}
		}
	}
	return nil
}

func validateSearchProviders(providers map[string]SearchProviderConfig, games map[string]GameConfig, stores map[string]StoreConfig) error {
	if len(providers) == 0 {
		return errors.New("invalid search provider configuration")
	}
	for name, provider := range providers {
		providerURL, err := url.ParseRequestURI(provider.URL)
		validType := provider.Type == "scrycl" || provider.Type == "tcgmatch" || provider.Type == "tcgdex"
		if name == "" || !validType || err != nil || providerURL.Scheme != "https" || providerURL.Host == "" || len(provider.Games) == 0 {
			return errors.New("invalid search provider configuration")
		}
		for _, game := range provider.Games {
			if _, known := games[game]; !known {
				return fmt.Errorf("unknown game %q for search provider %q", game, name)
			}
		}
		for _, store := range provider.Stores {
			if _, known := stores[store]; !known {
				return fmt.Errorf("unknown store %q for search provider %q", store, name)
			}
		}
	}
	return nil
}

// validateChecker Skips Stores that carry no Platform; they only list Inventory.
func validateChecker(store StoreConfig) error {
	if store.Platform == "" {
		return nil
	}
	validPlatform := store.Platform == "jumpseller" || store.Platform == "shopify" || store.Platform == "woocommerce" || store.Platform == "prestashop" || store.Platform == "html"
	if !validPlatform || store.TimeoutSeconds < 1 || store.EstimatedResponseSeconds < 0 {
		return errors.New("invalid store configuration")
	}
	return nil
}

func validateLists(lists []MoxfieldList) error {
	for _, list := range lists {
		if _, err := moxfield.ExtractListID(list.URL); err != nil {
			return err
		}
		if list.Label == "" || list.URL == "" || list.CLPPerCKUSD < 1 {
			return errors.New("invalid store configuration")
		}
	}
	return nil
}

// LanguageProviders Lists the Catalogs that can Answer a Game's Card Names. A
// Name Outside it Reaches no Adapter, so the Config Refuses it at Load.
var LanguageProviders = []string{"scryfall"}

func validateLanguages(catalog map[string]LanguageConfig, games map[string]GameConfig) error {
	for code, language := range catalog {
		if code == "" || language.Label == "" || language.Example == "" {
			return errors.New("invalid language catalog")
		}
	}
	for game, config := range games {
		declared := config.Languages
		if !declared.Declared() {
			continue
		}
		if !slices.Contains(LanguageProviders, declared.Provider) || len(declared.Supported) == 0 {
			return fmt.Errorf("invalid languages for game %q", game)
		}
		for _, code := range declared.Supported {
			if _, known := catalog[code]; !known {
				return fmt.Errorf("unknown language %q for game %q", code, game)
			}
		}
	}
	return nil
}
