package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Address       string
	DatabaseURL   string
	APIToken      string
	StoresPath    string
	ScryfallURL   string
	WorkerID      string
	LeaseDuration time.Duration
	PollInterval  time.Duration
	HTTPTimeout   time.Duration
}

func LoadConfig() (Config, error) {
	config := Config{
		Address: valueOr("MUCHI_API_ADDRESS", ":8080"), DatabaseURL: os.Getenv("DATABASE_URL"),
		APIToken: os.Getenv("MUCHI_API_TOKEN"), StoresPath: valueOr("MUCHI_STORES_CONFIG", "config/stores.yaml"),
		ScryfallURL: valueOr("MUCHI_SCRYFALL_URL", "https://api.scryfall.com"), WorkerID: valueOr("MUCHI_WORKER_ID", hostname()),
		LeaseDuration: secondsOr("MUCHI_LEASE_SECONDS", 60), PollInterval: secondsOr("MUCHI_POLL_SECONDS", 2), HTTPTimeout: secondsOr("MUCHI_SOURCE_TIMEOUT_SECONDS", 10),
	}
	if config.DatabaseURL == "" || config.APIToken == "" {
		return Config{}, errors.New("DATABASE_URL and MUCHI_API_TOKEN are required")
	}
	return config, nil
}

func valueOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func secondsOr(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}

func hostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "worker-local"
	}
	return value
}
