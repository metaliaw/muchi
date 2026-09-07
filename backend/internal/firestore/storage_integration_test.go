package firestore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

const testSearchTTL = 24 * time.Hour

func TestRecoverClaims(t *testing.T) {
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

func TestCancelIdempotency(t *testing.T) {
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

func TestExpireSearch(t *testing.T) {
	store := openTestStore(t)
	job := createTestSearch(t, store, "expiry")
	document, err := store.client.Collection("searches").Doc(job.ID).Get(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var record searchRecord
	if err := document.DataTo(&record); err != nil {
		t.Fatal(err)
	}
	ttl := time.Until(record.ExpiresAt)
	if ttl < 23*time.Hour || ttl > testSearchTTL {
		t.Fatalf("search ttl=%s", ttl)
	}
}

func TestCompleteItem(t *testing.T) {
	store := openTestStore(t)
	job := createTestSearch(t, store, "complete")
	item, err := store.ClaimSearchItem(context.Background(), "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	item.Status = search.ItemFound
	if err := store.CompleteSearchItem(context.Background(), item, nil); err != nil {
		t.Fatal(err)
	}
	updated, err := store.GetSearch(context.Background(), job.ID)
	if err != nil || updated.Processed != 1 || updated.Found != 1 {
		t.Fatalf("completed search=%#v err=%v", updated, err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	if os.Getenv("FIRESTORE_EMULATOR_HOST") == "" {
		t.Skip("FIRESTORE_EMULATOR_HOST is not set")
	}
	project := fmt.Sprintf("muchi-test-%d", time.Now().UnixNano())
	store, err := OpenStore(context.Background(), project, testSearchTTL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.CloseStore() })
	return store
}

func createTestSearch(t *testing.T, store *Store, suffix string) search.Job {
	t.Helper()
	input := search.CreateInput{
		Cards: []search.CardInput{
			{Name: "Sol Ring", Quantity: 1},
			{Name: "Anger", Quantity: 1},
		},
		Options: search.Options{VerifyStock: true, StoresOnly: true},
	}
	job, err := store.CreateSearch(context.Background(), "create-"+suffix, search.HashPayload(input), input)
	if err != nil {
		t.Fatal(err)
	}
	return job
}
