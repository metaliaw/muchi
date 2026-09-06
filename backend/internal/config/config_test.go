package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotenv(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("MUCHI_DOTENV_TEST=loaded\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	previous, found := os.LookupEnv("MUCHI_DOTENV_TEST")
	_ = os.Unsetenv("MUCHI_DOTENV_TEST")
	t.Cleanup(func() {
		if found {
			_ = os.Setenv("MUCHI_DOTENV_TEST", previous)
		} else {
			_ = os.Unsetenv("MUCHI_DOTENV_TEST")
		}
	})

	if err := loadDotenv(path); err != nil {
		t.Fatal(err)
	}
	if value := os.Getenv("MUCHI_DOTENV_TEST"); value != "loaded" {
		t.Fatalf("dotenv value=%q", value)
	}
}

func TestIgnoreMissingDotenv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.env")
	if err := loadDotenv(path); err != nil {
		t.Fatal(err)
	}
}
