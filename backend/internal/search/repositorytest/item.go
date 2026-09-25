package repositorytest

import "github.com/cangrejometralleta/muchi-api/internal/model"

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// RunItemLeaseContract checks two pending items through the repository port.
// Prepare must return a fresh repository containing exactly two pending items.
func RunItemLeaseContract(t *testing.T, prepare func(*testing.T) search.SearchItemRepository) {
	t.Helper()
	repository := prepare(t)
	ctx := context.Background()

	first, err := repository.ClaimSearchItem(ctx, "worker-one", time.Second)
	if err != nil || first.ID == "" || first.Status != model.ItemRunning {
		t.Fatalf("first claim=%+v err=%v", first, err)
	}
	if err := repository.RenewItemLease(ctx, first.ID, "worker-two", 3*time.Second); !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("another worker renewed the lease: %v", err)
	}
	if err := repository.RenewItemLease(ctx, first.ID, "worker-one", 3*time.Second); err != nil {
		t.Fatalf("owner could not renew the lease: %v", err)
	}
	time.Sleep(1200 * time.Millisecond)

	second, err := repository.ClaimSearchItem(ctx, "worker-two", 3*time.Second)
	if err != nil || second.ID == "" || second.ID == first.ID {
		t.Fatalf("second claim=%+v err=%v; first=%+v", second, err, first)
	}
	if _, err := repository.ClaimSearchItem(ctx, "worker-three", 3*time.Second); !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("renewed item was claimed again: %v", err)
	}

	first.Status = model.ItemFound
	if err := repository.CompleteSearchItem(ctx, first, nil); err != nil {
		t.Fatalf("owner could not complete the first item: %v", err)
	}
	second.Status = model.ItemFound
	if err := repository.CompleteSearchItem(ctx, second, nil); err != nil {
		t.Fatalf("owner could not complete the second item: %v", err)
	}
	if _, err := repository.ClaimSearchItem(ctx, "worker-three", 3*time.Second); !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("completed item was claimed again: %v", err)
	}
}
