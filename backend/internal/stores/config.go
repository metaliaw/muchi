package stores

import (
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"github.com/cangrejometralleta/muchi-api/internal/moxfield"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Games           map[string]GameConfig           `yaml:"games"`
	SearchProviders map[string]SearchProviderConfig `yaml:"search_providers"`
	Stores          map[string]StoreConfig          `yaml:"stores"`
}

type GameConfig struct {
	Name         string   `yaml:"name"`
	ReferenceKey string   `yaml:"reference_key"`
	Enabled      bool     `yaml:"enabled"`
	Origins      []string `yaml:"origins"`
}

type SearchProviderConfig struct {
	Type      string   `yaml:"type"`
	URL       string   `yaml:"url"`
	Enabled   bool     `yaml:"enabled"`
	Games     []string `yaml:"games"`
	Stores    []string `yaml:"stores"`
	Community bool     `yaml:"community"`
}

type StoreConfig struct {
	Games                []string       `yaml:"games"`
	Platform             string         `yaml:"platform"`
	UnavailableSelectors []string       `yaml:"unavailable_selectors"`
	UnavailableText      []string       `yaml:"unavailable_text"`
	ScopeSelector        string         `yaml:"scope_selector"`
	TimeoutSeconds       int            `yaml:"timeout_seconds"`
	AllowRedirects       bool           `yaml:"allow_redirects"`
	Enabled              bool           `yaml:"enabled"`
	Name                 string         `yaml:"name"`
	Lists                []MoxfieldList `yaml:"lists"`
}

// MoxfieldList Prices a public https://moxfield.com deck as store inventory.
type MoxfieldList struct {
	Label       string `yaml:"label"`
	URL         string `yaml:"url"`
	CLPPerCKUSD int    `yaml:"clp_per_ck_usd"`
}

func LoadStoreConfig(path string, logger *slog.Logger) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode store config: %w", err)
	}
	if err := ValidateConfig(config); err != nil {
		logStoreDebug(logger, "Store Config Rejected", "path", path, "error", err)
		return Config{}, err
	}
	logStoreDebug(logger, "Store Config Loaded", "path", path, "stores", len(config.Stores))
	return config, nil
}

// logStoreDebug Marks a Store Config State with muchi's one Cat Emoji.
func logStoreDebug(logger *slog.Logger, message string, args ...any) {
	if logger == nil {
		return
	}
	logger.Debug("🐱 "+message, args...)
}

func ValidateConfig(config Config) error {
	if err := validateGames(config.Games); err != nil {
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
		validType := provider.Type == "scry" || provider.Type == "tcgmatch"
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
	validPlatform := store.Platform == "jumpseller" || store.Platform == "shopify" || store.Platform == "woocommerce" || store.Platform == "html"
	if !validPlatform || store.TimeoutSeconds < 1 {
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
