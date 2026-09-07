package stores

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Stores map[string]StoreConfig `yaml:"stores"`
}

type StoreConfig struct {
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
	for domain, store := range config.Stores {
		if domain == "" {
			return errors.New("invalid store configuration")
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

// validateChecker Skips Stores that carry no Platform; they only list Inventory.
func validateChecker(store StoreConfig) error {
	if store.Platform == "" {
		return nil
	}
	validPlatform := store.Platform == "shopify" || store.Platform == "woocommerce" || store.Platform == "html"
	if !validPlatform || store.TimeoutSeconds < 1 {
		return errors.New("invalid store configuration")
	}
	return nil
}

func validateLists(lists []MoxfieldList) error {
	for _, list := range lists {
		if list.Label == "" || list.URL == "" || list.CLPPerCKUSD < 1 {
			return errors.New("invalid store configuration")
		}
	}
	return nil
}
