package gcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExistingGCPAdaptersRunTheSharedApplication(t *testing.T) {
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set")
	}
	path := filepath.Join(t.TempDir(), "stores.yaml")
	if err := os.WriteFile(path, []byte(`games:
  magic:
    name: Magic
    reference_key: magic
    origins: [scrycl]
search_providers:
  test:
    type: scrycl
    url: https://example.invalid
    games: [magic]
    enabled: false
stores: {}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", fmt.Sprintf("runtime-test-%d", time.Now().UnixNano()))
	t.Setenv("MUCHI_STORES_CONFIG", path)
	t.Setenv("MUCHI_API_TOKEN", "integration-token")
	t.Setenv("MUCHI_TASK_URL", "")
	t.Setenv("MUCHI_TASK_DEADLINE_SECONDS", "1800")
	t.Setenv("MUCHI_WORKER_ID", "integration-worker")
	instance, err := OpenInstance(context.Background(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := instance.Close(); err != nil {
			t.Error(err)
		}
	})
	request := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	response := httptest.NewRecorder()
	instance.Operations.API.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body)
	}
	request = httptest.NewRequest(http.MethodPost, "/v1/searches", strings.NewReader(`{"game":"magic","cards":[{"name":"Sol Ring","quantity":1}]}`))
	request.Header.Set("Authorization", "Bearer integration-token")
	request.Header.Set("Idempotency-Key", "runtime-create")
	response = httptest.NewRecorder()
	instance.Operations.API.ServeHTTP(response, request)
	var job struct {
		ID        string `json:"id"`
		Status    string `json:"status"`
		Processed int    `json:"processed"`
		Errors    int    `json:"errors"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusAccepted || job.ID == "" || job.Status != "queued" {
		t.Fatalf("create status=%d body=%s", response.Code, response.Body)
	}
	if err := instance.Operations.ProcessSearch(context.Background(), "cloud-task-owner"); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodGet, "/v1/searches/"+job.ID, nil)
	request.Header.Set("Authorization", "Bearer integration-token")
	response = httptest.NewRecorder()
	instance.Operations.API.ServeHTTP(response, request)
	if err := json.Unmarshal(response.Body.Bytes(), &job); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || job.Status != "completed_with_errors" || job.Processed != 1 || job.Errors != 1 {
		t.Fatalf("completed status=%d body=%s", response.Code, response.Body)
	}
}
