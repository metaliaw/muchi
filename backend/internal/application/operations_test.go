package application

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores/directory"
)

type operationVault struct {
	Vault
	events       []string
	owners       []string
	waiting      int
	claimError   error
	releaseError error
	releaseAge   time.Duration
	releaseLimit int
}

func (v *operationVault) CreateSearch(context.Context, string, string, model.CreateInput) (model.Job, error) {
	v.events = append(v.events, "persist")
	return model.Job{ID: "shared-search", Total: 1, Status: model.JobQueued}, nil
}
func (v *operationVault) ClaimSearchItem(ctx context.Context, owner string, _ time.Duration) (model.Item, error) {
	v.owners = append(v.owners, owner)
	if err := ctx.Err(); err != nil {
		return model.Item{}, err
	}
	if v.claimError != nil {
		return model.Item{}, v.claimError
	}
	return model.Item{}, search.ErrNotFound
}
func (v *operationVault) CountWaitingItems(_ context.Context, limit int) (int, error) {
	return min(v.waiting, limit), nil
}
func (v *operationVault) ReleaseExpiredOrders(_ context.Context, age time.Duration, limit int) (int, error) {
	v.releaseAge, v.releaseLimit = age, limit
	return 2, v.releaseError
}

type operationQueue struct {
	vault *operationVault
	wakes int
}

func (q *operationQueue) DispatchSearch(_ context.Context, job model.Job) error {
	if job.ID != "shared-search" {
		return errors.New("wrong search")
	}
	q.vault.events = append(q.vault.events, "dispatch")
	return nil
}
func (q *operationQueue) WakeWorker(context.Context, string) error {
	q.wakes++
	return nil
}

func TestInjectedAdaptersKeepPersistenceBeforeDispatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stores.yaml")
	if err := os.WriteFile(path, []byte(`games:
  magic:
    name: Magic
    reference_key: magic
    origins: [test]
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
	vault := &operationVault{}
	queue := &operationQueue{vault: vault}
	settings := config.Config{StoresPath: path, MaxCardsPerSearch: 10, MaxQuantityPerCard: 10}
	runtime, err := BuildRuntime(settings, nil, Dependencies{Vault: vault, Queue: queue})
	if err != nil {
		t.Fatal(err)
	}
	_, err = runtime.Service.CreateSearch(context.Background(), "request", model.CreateInput{
		Game: model.GameMagic, Cards: []model.CardInput{{Name: "Sol Ring", Quantity: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(vault.events, []string{"persist", "dispatch"}) {
		t.Fatalf("events=%v", vault.events)
	}
	runtime, err = BuildRuntime(settings, nil, Dependencies{Vault: vault})
	if err != nil {
		t.Fatal(err)
	}
	vault.events = nil
	_, err = runtime.Service.CreateSearch(context.Background(), "local", model.CreateInput{
		Game: model.GameMagic, Cards: []model.CardInput{{Name: "Sol Ring", Quantity: 1}},
	})
	if err != nil || !reflect.DeepEqual(vault.events, []string{"persist"}) {
		t.Fatalf("local events=%v err=%v", vault.events, err)
	}
}

func TestSharedHandlerKeepsStoresAuthenticationAndWebhookSecret(t *testing.T) {
	runtime := Runtime{Stores: []directory.ListedStore{{ID: "shared-store", Name: "Shared Store"}}}
	operations := BuildOperations(runtime, config.Config{APIToken: "api-secret", OrderWebhookSecret: "webhook-secret"}, nil)
	request := httptest.NewRequest(http.MethodGet, "/v1/stores", nil)
	response := httptest.NewRecorder()
	operations.API.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d", response.Code)
	}
	request.Header.Set("Authorization", "Bearer api-secret")
	response = httptest.NewRecorder()
	operations.API.ServeHTTP(response, request)
	var reply struct {
		Stores []directory.ListedStore `json:"stores"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(reply.Stores) != 1 || reply.Stores[0].ID != "shared-store" {
		t.Fatalf("status=%d reply=%+v", response.Code, reply)
	}
	body := `{"id":1,"status":"on-hold"}`
	signature := hmac.New(sha256.New, []byte("webhook-secret"))
	_, _ = signature.Write([]byte(body))
	request = httptest.NewRequest(http.MethodPost, "/v1/webhooks/woocommerce/store.example/orders", strings.NewReader(body))
	request.Header.Set("X-WC-Webhook-Signature", base64.StdEncoding.EncodeToString(signature.Sum(nil)))
	response = httptest.NewRecorder()
	operations.API.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("signed webhook status=%d body=%s", response.Code, response.Body)
	}
}

func TestOperationsKeepOwnerLocalToEachTurnAndPropagateCancellation(t *testing.T) {
	vault := &operationVault{}
	settings := config.Config{WorkerID: "local-worker"}
	operations := BuildOperations(Runtime{Vault: vault}, settings, nil)
	for _, owner := range []string{"cloud-task", ""} {
		if err := operations.ProcessSearch(context.Background(), owner); !errors.Is(err, search.ErrNotFound) {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(vault.owners, []string{"cloud-task", "local-worker"}) {
		t.Fatalf("owners=%v", vault.owners)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := operations.ProcessSearch(ctx, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

func TestMaintenanceUsesTheSamePoliciesAcrossHosts(t *testing.T) {
	vault := &operationVault{waiting: 5}
	queue := &operationQueue{vault: vault}
	settings := config.Config{SweepMaxWakes: 3, OrderPendingTTL: 24 * time.Hour, OrderReleaseMaxWakes: 7}
	operations := BuildOperations(Runtime{Vault: vault, Queue: queue}, settings, nil)
	if count, err := operations.SweepQueue(context.Background()); err != nil || count != 3 || queue.wakes != 3 {
		t.Fatalf("sweep count=%d wakes=%d err=%v", count, queue.wakes, err)
	}
	if count, err := operations.ReleaseOrders(context.Background()); err != nil || count != 2 {
		t.Fatalf("release=%d err=%v", count, err)
	}
	if vault.releaseAge != 24*time.Hour || vault.releaseLimit != 7 {
		t.Fatalf("release age=%s limit=%d", vault.releaseAge, vault.releaseLimit)
	}
	failure := errors.New("storage unavailable")
	vault.releaseError = failure
	if _, err := operations.ReleaseOrders(context.Background()); !errors.Is(err, failure) {
		t.Fatalf("error=%v", err)
	}
	operations = BuildOperations(Runtime{Vault: vault}, settings, nil)
	if _, err := operations.SweepQueue(context.Background()); err == nil {
		t.Fatal("sweep accepted missing dispatcher")
	}
}
