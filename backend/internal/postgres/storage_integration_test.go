package postgres

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

func TestClaimAndRecoverItems(t *testing.T) {
	store := openTestStore(t)
	job := createTestSearch(t, store, "claim")
	ctx := context.Background()
	var first, second search.Item
	var firstErr, secondErr error
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		first, firstErr = store.ClaimSearchItem(ctx, "worker-one", 50*time.Millisecond)
	}()
	go func() {
		defer group.Done()
		second, secondErr = store.ClaimSearchItem(ctx, "worker-two", time.Minute)
	}()
	group.Wait()
	if firstErr != nil || secondErr != nil || first.ID == second.ID {
		t.Fatalf("claims first=%#v/%v second=%#v/%v", first, firstErr, second, secondErr)
	}
	time.Sleep(75 * time.Millisecond)
	recovered, err := store.ClaimSearchItem(ctx, "worker-three", time.Minute)
	if err != nil || (recovered.ID != first.ID && recovered.ID != second.ID) {
		t.Fatalf("recover claim=%#v err=%v job=%s", recovered, err, job.ID)
	}
}

func TestCancelSearchIsIdempotent(t *testing.T) {
	store := openTestStore(t)
	job := createTestSearch(t, store, "cancel")
	first, err := store.CancelSearch(context.Background(), job.ID, "cancel-key", "same-hash")
	if err != nil || first.Status != search.JobCancelled {
		t.Fatalf("first cancel=%#v err=%v", first, err)
	}
	second, err := store.CancelSearch(context.Background(), job.ID, "cancel-key", "same-hash")
	if err != nil || second.Status != search.JobCancelled {
		t.Fatalf("second cancel=%#v err=%v", second, err)
	}
	_, err = store.CancelSearch(context.Background(), job.ID, "cancel-key", "other-hash")
	if !errors.Is(err, search.ErrConflict) {
		t.Fatalf("conflicting cancel error=%v", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("MUCHI_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("MUCHI_TEST_DATABASE_URL is not set")
	}
	store, err := OpenStore(dsn)
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.db.DB()
	if err != nil {
		t.Fatal(err)
	}
	down, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_initial.down.sql"))
	if err != nil {
		t.Fatal(err)
	}
	up, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000001_initial.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = db.Exec(string(down))
	if _, err := db.Exec(string(up)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(string(down)) })
	return store
}

func createTestSearch(t *testing.T, store *Store, suffix string) search.Job {
	t.Helper()
	input := search.CreateInput{Cards: []search.CardInput{{Name: "Sol Ring", Quantity: 1}, {Name: "Anger", Quantity: 1}}, Options: search.Options{VerifyStock: true, StoresOnly: true}}
	job, err := store.CreateSearch(context.Background(), "create-"+suffix, search.HashPayload(input), input)
	if err != nil {
		t.Fatal(err)
	}
	return job
}
