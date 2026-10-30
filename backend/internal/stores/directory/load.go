package directory

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

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
