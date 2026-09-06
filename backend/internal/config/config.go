package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Address       string
	ProjectID     string
	TaskRegion    string
	TaskQueue     string
	TaskURL       string
	TaskAccount   string
	APIToken      string
	StoresPath    string
	ScryfallURL   string
	WorkerID      string
	LeaseDuration time.Duration
	PollInterval  time.Duration
	HTTPTimeout   time.Duration
}

func LoadConfig() (Config, error) {
	if err := loadDotenv(".env"); err != nil {
		return Config{}, err
	}

	config := Config{
		Address:       readValueOr("MUCHI_API_ADDRESS", ":8080"),
		ProjectID:     os.Getenv("GOOGLE_CLOUD_PROJECT"),
		TaskRegion:    readValueOr("MUCHI_TASK_REGION", "us-central1"),
		TaskQueue:     readValueOr("MUCHI_TASK_QUEUE", "muchi-searches"),
		TaskURL:       os.Getenv("MUCHI_TASK_URL"),
		TaskAccount:   os.Getenv("MUCHI_TASK_SERVICE_ACCOUNT"),
		APIToken:      os.Getenv("MUCHI_API_TOKEN"),
		StoresPath:    readValueOr("MUCHI_STORES_CONFIG", "config/stores.yaml"),
		ScryfallURL:   readValueOr("MUCHI_SCRYFALL_URL", "https://api.scryfall.com"),
		WorkerID:      readValueOr("MUCHI_WORKER_ID", readHostname()),
		LeaseDuration: readSecondsOr("MUCHI_LEASE_SECONDS", 60),
		PollInterval:  readSecondsOr("MUCHI_POLL_SECONDS", 2),
		HTTPTimeout:   readSecondsOr("MUCHI_SOURCE_TIMEOUT_SECONDS", 10),
	}
	if config.APIToken == "" {
		return Config{}, errors.New("MUCHI_API_TOKEN is required")
	}
	return config, nil
}

func loadDotenv(path string) error {
	err := godotenv.Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func readValueOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func readSecondsOr(key string, fallback int) time.Duration {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		value = fallback
	}
	return time.Duration(value) * time.Second
}

func readHostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "worker-local"
	}
	return value
}
