package gcp

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Settings Holds GCP adapter configuration; application policies live in config.Config.
type Settings struct {
	ProjectID    string
	TaskRegion   string
	TaskQueue    string
	TaskURL      string
	TaskAccount  string
	TaskDeadline time.Duration
}

func LoadSettings() (Settings, error) {
	deadline := 30 * time.Minute
	if value := os.Getenv("MUCHI_TASK_DEADLINE_SECONDS"); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds < 15 || seconds > 1800 {
			return Settings{}, errors.New("MUCHI_TASK_DEADLINE_SECONDS must be between 15 and 1800")
		}
		deadline = time.Duration(seconds) * time.Second
	}
	return Settings{
		ProjectID:   os.Getenv("GOOGLE_CLOUD_PROJECT"),
		TaskRegion:  readSetting("MUCHI_TASK_REGION", "us-central1"),
		TaskQueue:   readSetting("MUCHI_TASK_QUEUE", "muchi-searches"),
		TaskURL:     os.Getenv("MUCHI_TASK_URL"),
		TaskAccount: os.Getenv("MUCHI_TASK_SERVICE_ACCOUNT"), TaskDeadline: deadline,
	}, nil
}

func readSetting(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
