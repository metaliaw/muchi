// Package db Implements persistence operations with Firebase Firestore.
package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	firestorelib "cloud.google.com/go/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
)

// maxRPCDeadline caps how far in the future a Firestore call's deadline can sit.
// Firestore rejects calls whose context deadline is more than ~30s out; callers
// (an HTTP handler, a Cloud Tasks-triggered function) may carry much longer ones.
const maxRPCDeadline = 20 * time.Second

// maxBatchWrites Bounds one Firestore Batch, and the Group one Bulk Read Asks for.
const maxBatchWrites = 500

// boundContext shortens ctx's deadline to maxRPCDeadline when the caller's is longer,
// while still honoring an earlier deadline or cancellation from the caller.
func boundContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= maxRPCDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, maxRPCDeadline)
}

// Store Holds the Firestore Client every Collection Reads and Writes through.
type Store struct {
	client *firestorelib.Client
}

type searchRecord struct {
	Payload   []byte    `firestore:"payload"`
	ItemIDs   []string  `firestore:"item_ids"`
	ExpiresAt time.Time `firestore:"expires_at"`
}

type itemRecord struct {
	Payload     []byte     `firestore:"payload"`
	SearchID    string     `firestore:"search_id"`
	Position    int        `firestore:"position"`
	Sequence    int        `firestore:"completion_sequence"`
	LeaseOwner  string     `firestore:"lease_owner"`
	LeaseUntil  *time.Time `firestore:"lease_until,omitempty"`
	AvailableAt time.Time  `firestore:"available_at"`
	ExpiresAt   time.Time  `firestore:"expires_at"`
}

type valueRecord struct {
	Payload      []byte    `firestore:"payload"`
	ExpiresAt    time.Time `firestore:"expires_at,omitempty"`
	OfferIndexed bool      `firestore:"offer_indexed,omitempty"`
}

type requestRecord struct {
	Hash       string    `firestore:"hash"`
	ResourceID string    `firestore:"resource_id"`
	ExpiresAt  time.Time `firestore:"expires_at"`
}

type orderRecord struct {
	Payload []byte `firestore:"payload"`
	Status  string `firestore:"status"`
	// Domain and StoreOrder are Indexed Copies of Fields the Payload already
	// Carries: FindOrderByStoreOrder Needs to Query by them, and a Payload is
	// opaque JSON to Firestore.
	Domain     string    `firestore:"domain"`
	StoreOrder string    `firestore:"store_order"`
	UpdatedAt  time.Time `firestore:"updated_at"`
}

type sourceRecord struct {
	Source              string     `firestore:"source"`
	LastSuccess         *time.Time `firestore:"last_success,omitempty"`
	LastFailure         *time.Time `firestore:"last_failure,omitempty"`
	ConsecutiveFailures int        `firestore:"consecutive_failures"`
	LatencyMilliseconds int64      `firestore:"latency_ms"`
	CircuitOpenUntil    *time.Time `firestore:"circuit_open_until,omitempty"`
}

// OpenStore Connects the Search Store through https://cloud.google.com/firestore/docs/reference/libraries.
func OpenStore(ctx context.Context, projectID string) (*Store, error) {
	client, err := firestorelib.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("open search store: %w", err)
	}
	return &Store{client: client}, nil
}

func (s *Store) CloseStore() error {
	return s.client.Close()
}

func packSearch(search repositories.Search) any {
	return searchRecord{Payload: mustJSON(renderJob(search.Job)), ItemIDs: search.ItemIDs, ExpiresAt: search.ExpiresAt}
}

// unpackSearch Answers a Document it cannot Read as Missing: a Search nobody
// can Read is a Search nobody Asked for.
func unpackSearch(document *firestorelib.DocumentSnapshot) (repositories.Search, error) {
	var record searchRecord
	if err := document.DataTo(&record); err != nil {
		return repositories.Search{}, repositories.ErrMissing
	}
	var job jobPayload
	err := json.Unmarshal(record.Payload, &job)
	return repositories.Search{Job: buildJob(job), ItemIDs: record.ItemIDs, ExpiresAt: record.ExpiresAt}, err
}

func packItem(stored repositories.Item) any {
	return itemRecord{
		Payload: mustJSON(renderItem(stored.Item)), SearchID: stored.SearchID, Position: stored.Position,
		Sequence: stored.Sequence, LeaseOwner: stored.LeaseOwner, LeaseUntil: stored.LeaseUntil,
		AvailableAt: stored.AvailableAt, ExpiresAt: stored.ExpiresAt,
	}
}

// unpackItem Marks a Document it cannot Read as Damaged instead of Failing:
// a Queue that Stops at one bad Document Stops for every Search behind it.
func unpackItem(document *firestorelib.DocumentSnapshot) (repositories.Item, error) {
	var record itemRecord
	if err := document.DataTo(&record); err != nil {
		return repositories.Item{Item: model.Item{ID: document.Ref.ID}, Damaged: true}, nil
	}
	var payload itemPayload
	err := json.Unmarshal(record.Payload, &payload)
	item := buildItem(payload)
	item.ID, item.LeaseOwner, item.LeaseUntil = document.Ref.ID, record.LeaseOwner, record.LeaseUntil
	return repositories.Item{Item: item, AvailableAt: record.AvailableAt, ExpiresAt: record.ExpiresAt, Damaged: err != nil}, nil
}

func packEntry(entry repositories.Entry) any {
	return valueRecord{Payload: mustJSON(renderOfferList(entry.Offers)), ExpiresAt: entry.ExpiresAt, OfferIndexed: entry.Indexed}
}

func unpackEntry(document *firestorelib.DocumentSnapshot) (repositories.Entry, error) {
	var record valueRecord
	if err := document.DataTo(&record); err != nil {
		return repositories.Entry{}, err
	}
	var payloads []offerPayload
	if err := json.Unmarshal(record.Payload, &payloads); err != nil {
		return repositories.Entry{}, err
	}
	return repositories.Entry{Offers: buildOfferList(payloads), ExpiresAt: record.ExpiresAt, Indexed: record.OfferIndexed}, nil
}

func packSnapshot(snapshot repositories.Snapshot) any {
	return valueRecord{Payload: mustJSON(renderOffer(snapshot.Offer)), ExpiresAt: snapshot.ExpiresAt}
}

func unpackSnapshot(document *firestorelib.DocumentSnapshot) (repositories.Snapshot, error) {
	var record valueRecord
	if err := document.DataTo(&record); err != nil {
		return repositories.Snapshot{}, err
	}
	var payload offerPayload
	if err := json.Unmarshal(record.Payload, &payload); err != nil {
		return repositories.Snapshot{}, err
	}
	return repositories.Snapshot{Offer: buildOffer(payload), ExpiresAt: record.ExpiresAt}, nil
}

func packRequest(request repositories.Request) any {
	return requestRecord{Hash: request.Hash, ResourceID: request.ResourceID, ExpiresAt: request.ExpiresAt}
}

func unpackRequest(document *firestorelib.DocumentSnapshot) (repositories.Request, error) {
	var record requestRecord
	err := document.DataTo(&record)
	return repositories.Request{Hash: record.Hash, ResourceID: record.ResourceID, ExpiresAt: record.ExpiresAt}, err
}

func packLease(lease repositories.Lease) any {
	return map[string]any{"next_allowed_at": lease.NextAllowedAt}
}

func unpackLease(document *firestorelib.DocumentSnapshot) (repositories.Lease, error) {
	next, err := document.DataAt("next_allowed_at")
	if err != nil {
		return repositories.Lease{}, err
	}
	stored, _ := next.(time.Time)
	return repositories.Lease{NextAllowedAt: stored}, nil
}

func packPing(repositories.Ping) any {
	return map[string]any{}
}

func unpackPing(*firestorelib.DocumentSnapshot) (repositories.Ping, error) {
	return repositories.Ping{}, nil
}

// packOrder Writes the Indexed Copies beside the Payload that the Store's
// Queries Filter on.
func packOrder(order model.Order) any {
	return orderRecord{Payload: mustJSON(renderOrder(order)), Status: string(order.Status), Domain: order.Domain, StoreOrder: order.StoreOrder, UpdatedAt: order.UpdatedAt}
}

func decodeOrder(document *firestorelib.DocumentSnapshot) (model.Order, error) {
	var record orderRecord
	if err := document.DataTo(&record); err != nil {
		return model.Order{}, err
	}
	var stored orderPayload
	err := json.Unmarshal(record.Payload, &stored)
	return buildOrder(stored), err
}

func packSourceHealth(health model.SourceHealth) any {
	return sourceRecord{
		Source: health.Source, LastSuccess: health.LastSuccess, LastFailure: health.LastFailure,
		ConsecutiveFailures: health.ConsecutiveFailures, LatencyMilliseconds: health.LatencyMilliseconds,
		CircuitOpenUntil: health.CircuitOpenUntil,
	}
}

func decodeSourceHealth(document *firestorelib.DocumentSnapshot) (model.SourceHealth, error) {
	var record sourceRecord
	err := document.DataTo(&record)
	return model.SourceHealth{
		Source: record.Source, LastSuccess: record.LastSuccess, LastFailure: record.LastFailure,
		ConsecutiveFailures: record.ConsecutiveFailures,
		LatencyMilliseconds: record.LatencyMilliseconds,
		CircuitOpenUntil:    record.CircuitOpenUntil,
	}, err
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
