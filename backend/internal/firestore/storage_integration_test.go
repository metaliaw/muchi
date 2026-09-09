package firestore

import (
	"context"
	"encoding/json"
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

func TestListEmptySources(t *testing.T) {
	store := openTestStore(t)
	items, err := store.ListSourceHealth(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if items == nil || len(items) != 0 {
		t.Fatalf("empty sources=%#v", items)
	}

	body, err := json.Marshal(map[string]any{"sources": items})
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"sources":[]}` {
		t.Fatalf("empty sources JSON=%s", body)
	}
}

// TestPoisonItemDoesNotBlockTheQueue Reproduces the Stall that stopped every
// Search: one Document that a Claim always refuses, sitting at the Head.
func TestPoisonItemDoesNotBlockTheQueue(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	job := createTestSearch(t, store, "poison")

	// The Poison: expired, and available before every healthy Item, so the
	// Query hands it over first on every single Turn.
	poison := itemRecord{
		Payload:     mustJSON(search.Item{ID: "item_poison", SearchID: job.ID, Status: search.ItemPending}),
		SearchID:    job.ID,
		AvailableAt: time.Now().UTC().Add(-time.Hour),
		ExpiresAt:   time.Now().UTC().Add(-time.Hour),
	}
	if _, err := store.client.Collection("items").Doc("item_poison").Set(ctx, poison); err != nil {
		t.Fatal(err)
	}

	claimed, err := store.ClaimSearchItem(ctx, "worker", time.Minute)
	if err != nil {
		t.Fatalf("the Queue stalled behind the poisoned Item: %v", err)
	}
	if claimed.ID == "item_poison" {
		t.Fatalf("an expired Item was claimed: %#v", claimed)
	}
}

// TestFinishedItemLeavesTheQueue Keeps dead Documents out of the Candidates.
// Parked on its own Expiry, a finished Item came back as an eternal Skip.
func TestFinishedItemLeavesTheQueue(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	createTestSearch(t, store, "finished")

	item, err := store.ClaimSearchItem(ctx, "worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	item.Status = search.ItemFound
	if err := store.CompleteSearchItem(ctx, item, nil); err != nil {
		t.Fatal(err)
	}

	document, err := store.client.Collection("items").Doc(item.ID).Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var record itemRecord
	if err := document.DataTo(&record); err != nil {
		t.Fatal(err)
	}
	if !record.AvailableAt.After(record.ExpiresAt) {
		t.Fatalf("a finished Item stays claimable at its Expiry: available=%v expires=%v",
			record.AvailableAt, record.ExpiresAt)
	}
}

// TestCountWaitingItemsIgnoresDeadDocuments Keeps the Sweeper honest. Counting
// a dead Document would ask for Turns that only ever skip it.
func TestCountWaitingItemsIgnoresDeadDocuments(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	job := createTestSearch(t, store, "waiting")

	waiting, err := store.CountWaitingItems(ctx, 50)
	if err != nil || waiting != 2 {
		t.Fatalf("two fresh Items must wait: waiting=%d err=%v", waiting, err)
	}

	dead := itemRecord{
		Payload:     mustJSON(search.Item{ID: "item_dead", SearchID: job.ID, Status: search.ItemPending}),
		SearchID:    job.ID,
		AvailableAt: time.Now().UTC().Add(-time.Hour),
		ExpiresAt:   time.Now().UTC().Add(-time.Hour),
	}
	if _, err := store.client.Collection("items").Doc("item_dead").Set(ctx, dead); err != nil {
		t.Fatal(err)
	}

	waiting, err = store.CountWaitingItems(ctx, 50)
	if err != nil || waiting != 2 {
		t.Fatalf("a dead Document is not Work: waiting=%d err=%v", waiting, err)
	}
}

// TestCountWaitingItemsRespectsTheCap Stops one Sweep from firing everything.
func TestCountWaitingItemsRespectsTheCap(t *testing.T) {
	store := openTestStore(t)
	createTestSearch(t, store, "capped")

	waiting, err := store.CountWaitingItems(context.Background(), 1)
	if err != nil || waiting != 1 {
		t.Fatalf("waiting=%d err=%v", waiting, err)
	}
}
