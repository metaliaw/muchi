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

func TestTaskDeadline(t *testing.T) {
	t.Setenv("MUCHI_API_TOKEN", "test")
	for _, test := range []struct {
		value string
		valid bool
	}{{"900", true}, {"15", true}, {"1800", true}, {"14", false}, {"1801", false}, {"0", false}, {"-1", false}, {"abc", false}, {"9999999999999999999999", false}} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("MUCHI_TASK_DEADLINE_SECONDS", test.value)
			_, err := LoadConfig()
			if (err == nil) != test.valid {
				t.Fatalf("deadline=%s err=%v", test.value, err)
			}
		})
	}
}
