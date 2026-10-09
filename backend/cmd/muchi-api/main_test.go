package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/application"
	"github.com/cangrejometralleta/muchi-api/internal/config"
	"github.com/cangrejometralleta/muchi-api/internal/hosting"
)

type commandVault struct {
	application.Vault
	age   time.Duration
	limit int
}

func (v *commandVault) ReleaseExpiredOrders(_ context.Context, age time.Duration, limit int) (int, error) {
	v.age, v.limit = age, limit
	return 1, nil
}

func TestCommandReleaseCallsSharedOperation(t *testing.T) {
	vault := &commandVault{}
	settings := config.Config{OrderPendingTTL: 24 * time.Hour, OrderReleaseMaxWakes: 6}
	instance := hosting.Instance{Settings: settings, Operations: application.BuildOperations(application.Runtime{Vault: vault}, settings, nil)}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := executeCommand(context.Background(), logger, "release", instance); err != nil {
		t.Fatal(err)
	}
	if vault.age != 24*time.Hour || vault.limit != 6 {
		t.Fatalf("age=%s limit=%d", vault.age, vault.limit)
	}
}

func TestUnknownCommandDoesNotInitializeCloudResources(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, args := range [][]string{nil, {"unknown"}, {"serve", "extra"}} {
		if err := runCommand(logger, args); err == nil {
			t.Fatalf("accepted args=%v", args)
		}
	}
}

func TestContainerServerKeepsTheSharedHTTPHandler(t *testing.T) {
	operations := application.BuildOperations(application.Runtime{}, config.Config{APIToken: "shared-token"}, nil)
	settings := config.Config{Address: ":8080", ServerReadHeaderTimeout: time.Second}
	server := buildAPIServer(settings, operations.API)
	request := httptest.NewRequest(http.MethodGet, "/v1/stores", nil)
	request.Header.Set("Authorization", "Bearer shared-token")
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "{\"stores\":[]}\n" {
		t.Fatalf("status=%d body=%s", response.Code, response.Body)
	}
	if server.Addr != ":8080" || server.ReadHeaderTimeout != time.Second {
		t.Fatalf("server=%+v", server)
	}
}

func TestShutdownWaitsForAnActiveRequest(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	response := make(chan error, 1)
	go func() {
		reply, err := server.Client().Get(server.URL)
		if err == nil {
			err = reply.Body.Close()
		}
		response <- err
	}()
	<-entered
	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan error, 1)
	go func() { stopped <- stopServer(ctx, server.Config, 5*time.Second) }()
	cancel()
	select {
	case err := <-stopped:
		close(release)
		t.Fatalf("shutdown finished with an active request: %v", err)
	default:
	}
	close(release)
	if err := <-response; err != nil {
		t.Fatal(err)
	}
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
}
