package gcp

import (
	"testing"
	"time"
)

func TestTaskDeadline(t *testing.T) {
	for _, test := range []struct {
		value string
		valid bool
	}{{"900", true}, {"15", true}, {"1800", true}, {"14", false}, {"1801", false}, {"0", false}, {"-1", false}, {"abc", false}, {"9999999999999999999999", false}} {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("MUCHI_TASK_DEADLINE_SECONDS", test.value)
			_, err := LoadSettings()
			if (err == nil) != test.valid {
				t.Fatalf("deadline=%s err=%v", test.value, err)
			}
		})
	}
}

func TestExistingCloudEnvironmentStillSelectsAdapters(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "existing-project")
	t.Setenv("MUCHI_TASK_REGION", "southamerica-east1")
	t.Setenv("MUCHI_TASK_QUEUE", "existing-queue")
	t.Setenv("MUCHI_TASK_URL", "https://worker.example")
	t.Setenv("MUCHI_TASK_SERVICE_ACCOUNT", "tasks@example")
	t.Setenv("MUCHI_TASK_DEADLINE_SECONDS", "1800")
	settings, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.ProjectID != "existing-project" || settings.TaskRegion != "southamerica-east1" ||
		settings.TaskQueue != "existing-queue" || settings.TaskURL != "https://worker.example" ||
		settings.TaskAccount != "tasks@example" || settings.TaskDeadline != 30*time.Minute {
		t.Fatalf("settings=%+v", settings)
	}
}
