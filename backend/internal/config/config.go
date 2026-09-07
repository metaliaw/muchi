package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Address                 string
	ProjectID               string
	TaskRegion              string
	TaskQueue               string
	TaskURL                 string
	TaskAccount             string
	TaskDeadline            time.Duration
	APIToken                string
	StoresPath              string
	ScryURL                 string
	ScryEnabled             bool
	WorkerID                string
	LeaseDuration           time.Duration
	PollInterval            time.Duration
	HTTPTimeout             time.Duration
	SourceMaxAttempts       int
	SourceRetryBaseDelay    time.Duration
	SourceMaxBodyBytes      int64
	ServerReadHeaderTimeout time.Duration
	ServerReadTimeout       time.Duration
	ServerWriteTimeout      time.Duration
	ServerIdleTimeout       time.Duration
	ServerShutdownTimeout   time.Duration
	HealthCheckTimeout      time.Duration
	StockCheckLimit         int
	OfferCacheTTL           time.Duration
	OfferCacheEmptyTTL      time.Duration
	SearchTTL               time.Duration
	MaxCardsPerSearch       int
	MaxQuantityPerCard      int
	SuspiciousPricePercent  int
}

func LoadConfig() (Config, error) {
	if err := loadDotenv(".env"); err != nil {
		return Config{}, err
	}

	deadline, err := readTaskDeadline()
	if err != nil {
		return Config{}, err
	}
	config := Config{
		Address:                 readValueOr("MUCHI_API_ADDRESS", ":8080"),
		ProjectID:               os.Getenv("GOOGLE_CLOUD_PROJECT"),
		TaskRegion:              readValueOr("MUCHI_TASK_REGION", "us-central1"),
		TaskQueue:               readValueOr("MUCHI_TASK_QUEUE", "muchi-searches"),
		TaskURL:                 os.Getenv("MUCHI_TASK_URL"),
		TaskAccount:             os.Getenv("MUCHI_TASK_SERVICE_ACCOUNT"),
		TaskDeadline:            deadline,
		APIToken:                os.Getenv("MUCHI_API_TOKEN"),
		StoresPath:              readValueOr("MUCHI_STORES_CONFIG", "config/stores.yaml"),
		ScryURL:                 readValueOr("MUCHI_SCRY_URL", "https://scry.cl"),
		ScryEnabled:             readBoolOr("MUCHI_SCRY_ENABLED", true),
		WorkerID:                readValueOr("MUCHI_WORKER_ID", readHostname()),
		LeaseDuration:           readSecondsOr("MUCHI_LEASE_SECONDS", 60),
		PollInterval:            readSecondsOr("MUCHI_POLL_SECONDS", 2),
		HTTPTimeout:             readSecondsOr("MUCHI_SOURCE_TIMEOUT_SECONDS", 10),
		SourceMaxAttempts:       readIntOr("MUCHI_SOURCE_MAX_ATTEMPTS", 3),
		SourceRetryBaseDelay:    readMillisOr("MUCHI_SOURCE_RETRY_BASE_MS", 250),
		SourceMaxBodyBytes:      int64(readIntOr("MUCHI_SOURCE_MAX_BODY_BYTES", 4<<20)),
		ServerReadHeaderTimeout: readSecondsOr("MUCHI_SERVER_READ_HEADER_TIMEOUT_SECONDS", 5),
		ServerReadTimeout:       readSecondsOr("MUCHI_SERVER_READ_TIMEOUT_SECONDS", 15),
		ServerWriteTimeout:      readSecondsOr("MUCHI_SERVER_WRITE_TIMEOUT_SECONDS", 30),
		ServerIdleTimeout:       readSecondsOr("MUCHI_SERVER_IDLE_TIMEOUT_SECONDS", 60),
		ServerShutdownTimeout:   readSecondsOr("MUCHI_SERVER_SHUTDOWN_TIMEOUT_SECONDS", 10),
		HealthCheckTimeout:      readSecondsOr("MUCHI_HEALTH_CHECK_TIMEOUT_SECONDS", 2),
		StockCheckLimit:         readIntOr("MUCHI_STOCK_CHECK_LIMIT", 5),
		OfferCacheTTL:           readSecondsOr("MUCHI_OFFER_CACHE_TTL_SECONDS", 900),
		OfferCacheEmptyTTL:      readSecondsOr("MUCHI_OFFER_CACHE_EMPTY_TTL_SECONDS", 120),
		SearchTTL:               readSecondsOr("MUCHI_SEARCH_TTL_SECONDS", 86400),
		MaxCardsPerSearch:       readIntOr("MUCHI_MAX_CARDS_PER_SEARCH", 500),
		MaxQuantityPerCard:      readIntOr("MUCHI_MAX_QUANTITY_PER_CARD", 99),
		SuspiciousPricePercent:  readIntOr("MUCHI_SUSPICIOUS_PRICE_PERCENT", 30),
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

func readIntOr(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

// readBoolOr Reads a Flag, Falling Back when Unset or Unreadable.
func readBoolOr(key string, fallback bool) bool {
	value, err := strconv.ParseBool(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return value
}

func readSecondsOr(key string, fallback int) time.Duration {
	return time.Duration(readIntOr(key, fallback)) * time.Second
}

func readMillisOr(key string, fallback int) time.Duration {
	return time.Duration(readIntOr(key, fallback)) * time.Millisecond
}

func readHostname() string {
	value, err := os.Hostname()
	if err != nil {
		return "worker-local"
	}
	return value
}

func readTaskDeadline() (time.Duration, error) {
	value := os.Getenv("MUCHI_TASK_DEADLINE_SECONDS")
	if value == "" {
		return 30 * time.Minute, nil
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 15 || seconds > 1800 {
		return 0, errors.New("MUCHI_TASK_DEADLINE_SECONDS must be between 15 and 1800")
	}
	return time.Duration(seconds) * time.Second, nil
}
