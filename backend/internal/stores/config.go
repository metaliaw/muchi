package stores

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Stores map[string]StoreConfig `yaml:"stores"`
}

type StoreConfig struct {
	Platform             string   `yaml:"platform"`
	UnavailableSelectors []string `yaml:"unavailable_selectors"`
	UnavailableText      []string `yaml:"unavailable_text"`
	ScopeSelector        string   `yaml:"scope_selector"`
	TimeoutSeconds       int      `yaml:"timeout_seconds"`
	AllowRedirects       bool     `yaml:"allow_redirects"`
	Enabled              bool     `yaml:"enabled"`
}

func LoadStoreConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("decode store config: %w", err)
	}
	return config, ValidateConfig(config)
}

func ValidateConfig(config Config) error {
	for domain, store := range config.Stores {
		validPlatform := store.Platform == "shopify" || store.Platform == "woocommerce" || store.Platform == "html"
		if domain == "" || !validPlatform || store.TimeoutSeconds < 1 {
			return errors.New("invalid store configuration")
		}
	}
	return nil
}
