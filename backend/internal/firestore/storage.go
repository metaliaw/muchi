package firestore

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	firestorelib "cloud.google.com/go/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxRPCDeadline caps how far in the future a Firestore call's deadline can sit.
// Firestore rejects calls whose context deadline is more than ~30s out; callers
// (an HTTP handler, a Cloud Tasks-triggered function) may carry much longer ones.
const maxRPCDeadline = 20 * time.Second

// One Turn discards this many dead Items before its final Claim. The Bound
// keeps a damaged Queue from holding one Function open without Limit.
const maxClaimDiscards = 20

// A finished Item is parked beyond its own Expiry. Parked exactly on it, it
// became claimable at the very Instant a Claim starts refusing it, and every
// Turn from then on wasted a Candidate Slot on a dead Document.
const doneParking = 24 * time.Hour

// boundContext shortens ctx's deadline to maxRPCDeadline when the caller's is longer,
// while still honoring an earlier deadline or cancellation from the caller.
func boundContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) <= maxRPCDeadline {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, maxRPCDeadline)
}

type Store struct {
	client    *firestorelib.Client
	searchTTL time.Duration
	logger    *slog.Logger
}

// TellStore Gives the Store a Voice. Without one it stays silent, which is
// what every Test wants and what Production must never have.
func (s *Store) TellStore(logger *slog.Logger) {
	s.logger = logger
}

func (s *Store) say() *slog.Logger {
	if s.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return s.logger
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
	LeaseOwner  string     `firestore:"lease_owner"`
	LeaseUntil  *time.Time `firestore:"lease_until,omitempty"`
	AvailableAt time.Time  `firestore:"available_at"`
	ExpiresAt   time.Time  `firestore:"expires_at"`
}

type valueRecord struct {
	Payload   []byte    `firestore:"payload"`
	ExpiresAt time.Time `firestore:"expires_at,omitempty"`
}

type requestRecord struct {
	Hash       string    `firestore:"hash"`
	ResourceID string    `firestore:"resource_id"`
	ExpiresAt  time.Time `firestore:"expires_at"`
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
func OpenStore(ctx context.Context, projectID string, searchTTL time.Duration) (*Store, error) {
	client, err := firestorelib.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("open search store: %w", err)
	}
	return &Store{client: client, searchTTL: searchTTL}, nil
}

func (s *Store) CreateSearch(ctx context.Context, key, hash string, input search.CreateInput) (search.Job, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	var job search.Job
	var created bool
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		stored, found, err := s.readRequest(ctx, tx, "create", key)
		if err != nil {
			return err
		}
		if found {
			if stored.Hash != hash {
				return search.ErrConflict
			}
			job, err = s.readSearch(ctx, tx, stored.ResourceID)
			return err
		}
		job = buildSearch(input)
		created = true
		return s.createSearch(ctx, tx, key, hash, job)
	})
	if err != nil || !created {
		return job, mapStoreError(err)
	}
	return job, s.createItems(ctx, job, input)
}

func (s *Store) GetSearch(ctx context.Context, id string) (search.Job, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	document, err := s.client.Collection("searches").Doc(id).Get(ctx)
	if err != nil {
		return search.Job{}, mapStoreError(err)
	}
	return decodeSearch(document)
}

func (s *Store) ListResults(ctx context.Context, id string) (search.Result, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	document, err := s.client.Collection("searches").Doc(id).Get(ctx)
	if err != nil {
		return search.Result{}, mapStoreError(err)
	}
	var record searchRecord
	if err := document.DataTo(&record); err != nil || record.ExpiresAt.Before(time.Now()) {
		return search.Result{}, search.ErrNotFound
	}
	items, err := s.loadItems(ctx, record.ItemIDs)
	return search.Result{SearchID: id, Items: items}, err
}

func (s *Store) CancelSearch(ctx context.Context, id, key, hash string) (search.Job, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	var job search.Job
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		stored, found, err := s.readRequest(ctx, tx, "cancel", key)
		if err != nil {
			return err
		}
		if found {
			if stored.Hash != hash {
				return search.ErrConflict
			}
			job, err = s.readSearch(ctx, tx, stored.ResourceID)
			return err
		}
		job, err = s.readSearch(ctx, tx, id)
		if err != nil {
			return err
		}
		return s.cancelSearch(ctx, tx, key, hash, &job)
	})
	return job, mapStoreError(err)
}

func (s *Store) ClaimSearchItem(ctx context.Context, owner string, lease time.Duration) (search.Item, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	for attempt := 0; attempt <= maxClaimDiscards; attempt++ {
		claimed, discarded, err := s.claimSearchItem(ctx, owner, lease)
		if err != nil || claimed.ID != "" {
			return claimed, mapStoreError(err)
		}
		s.say().Warn("Claim Discarded Dead Item", "item", discarded)
	}
	return search.Item{}, search.ErrNotFound
}

func (s *Store) claimSearchItem(ctx context.Context, owner string, lease time.Duration) (search.Item, string, error) {
	var claimed search.Item
	var discarded string
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		query := s.client.Collection("items").
			Where("available_at", "<=", time.Now().UTC()).
			OrderBy("available_at", firestorelib.Asc).
			Limit(1)
		documents := tx.Documents(query)
		for {
			document, err := documents.Next()
			if errors.Is(err, iterator.Done) {
				return search.ErrNotFound
			}
			if err != nil {
				return err
			}
			// A Candidate that cannot be claimed is skipped, never fatal.
			// claimItem refuses before it writes, so the Transaction stays
			// clean until one of them is actually taken.
			err = s.claimItem(ctx, tx, document, owner, lease, &claimed)
			if errors.Is(err, search.ErrNotFound) {
				discarded = document.Ref.ID
				return tx.Delete(document.Ref)
			}
			return err
		}
	})
	return claimed, discarded, err
}

// CountWaitingItems Says how many Items are ready for a Turn right now.
//
// A Task carries no Item: it means "wake up and take whatever is there", so
// the only Bond between Work and Wake-ups is the Count. A Turn that ends
// without working spends a Wake-up and nothing puts it back; the Sweeper
// compares this Count with the Queue and repairs the Difference.
func (s *Store) CountWaitingItems(ctx context.Context, limit int) (int, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	now := time.Now().UTC()
	documents := s.client.Collection("items").
		Where("available_at", "<=", now).
		OrderBy("available_at", firestorelib.Asc).
		Limit(limit).
		Documents(ctx)
	defer documents.Stop()
	waiting := 0
	for {
		document, err := documents.Next()
		if errors.Is(err, iterator.Done) {
			return waiting, nil
		}
		if err != nil {
			return waiting, mapStoreError(err)
		}
		var record itemRecord
		// A dead Document is not Work: counting it would ask for Turns that
		// only ever skip it.
		if err := document.DataTo(&record); err != nil || record.ExpiresAt.Before(now) {
			continue
		}
		waiting++
	}
}

func (s *Store) RenewItemLease(ctx context.Context, id, owner string, lease time.Duration) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		reference := s.client.Collection("items").Doc(id)
		document, err := tx.Get(reference)
		if err != nil {
			return mapStoreError(err)
		}
		item, record, err := decodeItem(document)
		if err != nil || item.LeaseOwner != owner || item.Status != search.ItemRunning {
			return search.ErrNotFound
		}
		expires := time.Now().UTC().Add(lease)
		item.LeaseUntil, record.AvailableAt = &expires, expires
		record.LeaseUntil = &expires
		record.Payload, _ = json.Marshal(item)
		return tx.Set(reference, record)
	})
}

func (s *Store) CompleteSearchItem(ctx context.Context, item search.Item, offers []offer.Offer) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		itemReference := s.client.Collection("items").Doc(item.ID)
		document, err := tx.Get(itemReference)
		if err != nil {
			return mapStoreError(err)
		}
		current, record, err := decodeItem(document)
		if err != nil || current.LeaseOwner != item.LeaseOwner || current.Status != search.ItemRunning {
			return search.ErrNotFound
		}
		return s.completeItem(ctx, tx, itemReference, record, item, offers)
	})
}

func (s *Store) LoadOffers(ctx context.Context, key string) ([]offer.Offer, bool, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	document, err := s.client.Collection("offer_cache").Doc(hashKey(key)).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var record valueRecord
	if err := document.DataTo(&record); err != nil {
		return nil, false, err
	}
	if record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, false, nil
	}
	var items []offer.Offer
	err = json.Unmarshal(record.Payload, &items)
	return items, err == nil, err
}

func (s *Store) SaveOffers(ctx context.Context, key string, items []offer.Offer, ttl time.Duration) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	record := valueRecord{Payload: data, ExpiresAt: time.Now().UTC().Add(ttl)}
	_, err = s.client.Collection("offer_cache").Doc(hashKey(key)).Set(ctx, record)
	return err
}

func (s *Store) AwaitSource(ctx context.Context, domain string) error {
	boundCtx, cancel := boundContext(ctx)
	defer cancel()
	record, _ := s.loadSource(boundCtx, domain)
	if record.CircuitOpenUntil != nil && record.CircuitOpenUntil.After(time.Now().UTC()) {
		return fmt.Errorf("%w until %s", source.ErrCircuitOpen, record.CircuitOpenUntil.Format(time.RFC3339))
	}
	allowed, err := s.reserveTraffic(boundCtx, domain)
	if err != nil {
		return err
	}
	return waitUntil(ctx, allowed)
}

func (s *Store) RecordSource(ctx context.Context, domain string, latency time.Duration, sourceErr error) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	reference := s.client.Collection("source_health").Doc(hashKey(domain))
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		record, _ := s.readSource(ctx, tx, reference, domain)
		updateSource(&record, latency, sourceErr)
		return tx.Set(reference, record)
	})
}

func (s *Store) CheckHealth(ctx context.Context) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	_, err := s.client.Collection("health").Doc("ping").Get(ctx)
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

func (s *Store) ListSourceHealth(ctx context.Context) ([]search.SourceHealth, error) {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	documents := s.client.Collection("source_health").Documents(ctx)
	defer documents.Stop()
	result := make([]search.SourceHealth, 0)
	for {
		document, err := documents.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		var record sourceRecord
		if err := document.DataTo(&record); err != nil {
			return nil, err
		}
		result = append(result, mapSourceHealth(record))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Source < result[j].Source })
	return result, nil
}

func (s *Store) CloseStore() error {
	return s.client.Close()
}

func (s *Store) createSearch(ctx context.Context, tx *firestorelib.Transaction, key, hash string, job search.Job) error {
	itemIDs := make([]string, job.Total)
	for index := range itemIDs {
		itemIDs[index] = buildID("item")
	}
	expires := time.Now().UTC().Add(s.searchTTL)
	data, _ := json.Marshal(job)
	if err := tx.Create(s.client.Collection("searches").Doc(job.ID), searchRecord{Payload: data, ItemIDs: itemIDs, ExpiresAt: expires}); err != nil {
		return err
	}
	record := requestRecord{Hash: hash, ResourceID: job.ID, ExpiresAt: expires}
	return tx.Set(s.client.Collection("idempotency").Doc(hashKey("create:"+key)), record)
}

func (s *Store) createItems(ctx context.Context, job search.Job, input search.CreateInput) error {
	document, err := s.client.Collection("searches").Doc(job.ID).Get(ctx)
	if err != nil {
		return err
	}
	var stored searchRecord
	if err := document.DataTo(&stored); err != nil {
		return err
	}
	batch := s.client.Batch()
	for index, card := range input.Cards {
		item := search.Item{
			ID: stored.ItemIDs[index], SearchID: job.ID, Position: index,
			OriginalName: card.Name, NormalizedName: offer.NormalizeCard(card.Name),
			Quantity: card.Quantity, Status: search.ItemPending, Offers: []offer.Offer{},
			VerifyStock: input.Options.VerifyStock, StoresOnly: input.Options.StoresOnly,
		}
		data, _ := json.Marshal(item)
		record := itemRecord{Payload: data, SearchID: job.ID, Position: index, AvailableAt: job.CreatedAt, ExpiresAt: stored.ExpiresAt}
		batch.Set(s.client.Collection("items").Doc(item.ID), record)
	}
	_, err = batch.Commit(ctx)
	return err
}

func (s *Store) readRequest(ctx context.Context, tx *firestorelib.Transaction, action, key string) (requestRecord, bool, error) {
	document, err := tx.Get(s.client.Collection("idempotency").Doc(hashKey(action + ":" + key)))
	if status.Code(err) == codes.NotFound {
		return requestRecord{}, false, nil
	}
	if err != nil {
		return requestRecord{}, false, err
	}
	var record requestRecord
	if err := document.DataTo(&record); err != nil {
		return requestRecord{}, false, err
	}
	return record, !record.ExpiresAt.Before(time.Now().UTC()), nil
}

func (s *Store) readSearch(ctx context.Context, tx *firestorelib.Transaction, id string) (search.Job, error) {
	document, err := tx.Get(s.client.Collection("searches").Doc(id))
	if err != nil {
		return search.Job{}, mapStoreError(err)
	}
	return decodeSearch(document)
}

func (s *Store) cancelSearch(ctx context.Context, tx *firestorelib.Transaction, key, hash string, job *search.Job) error {
	if job.Status != search.JobQueued && job.Status != search.JobRunning {
		return search.ErrNotRunning
	}
	now := time.Now().UTC()
	job.Status, job.FinishedAt, job.UpdatedAt = search.JobCancelled, &now, now
	data, _ := json.Marshal(job)
	reference := s.client.Collection("searches").Doc(job.ID)
	if err := tx.Update(reference, []firestorelib.Update{{Path: "payload", Value: data}}); err != nil {
		return err
	}
	record := requestRecord{Hash: hash, ResourceID: job.ID, ExpiresAt: now.Add(s.searchTTL)}
	return tx.Set(s.client.Collection("idempotency").Doc(hashKey("cancel:"+key)), record)
}

func (s *Store) claimItem(ctx context.Context, tx *firestorelib.Transaction, document *firestorelib.DocumentSnapshot, owner string, lease time.Duration, claimed *search.Item) error {
	item, record, err := decodeItem(document)
	if err != nil || record.ExpiresAt.Before(time.Now().UTC()) {
		return search.ErrNotFound
	}
	job, err := s.readSearch(ctx, tx, item.SearchID)
	if status.Code(err) == codes.NotFound || errors.Is(err, search.ErrNotFound) || job.Status == search.JobCancelled {
		return search.ErrNotFound
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	expires := now.Add(lease)
	item.Status, item.Attempts, item.LeaseOwner, item.LeaseUntil = search.ItemRunning, item.Attempts+1, owner, &expires
	record.Payload, record.AvailableAt = mustJSON(item), expires
	record.LeaseOwner, record.LeaseUntil = owner, &expires
	job.Status, job.CurrentCard, job.UpdatedAt = search.JobRunning, item.OriginalName, now
	if job.StartedAt == nil {
		job.StartedAt = &now
	}
	if err := tx.Set(document.Ref, record); err != nil {
		return err
	}
	if err := tx.Update(s.client.Collection("searches").Doc(job.ID), []firestorelib.Update{{Path: "payload", Value: mustJSON(job)}}); err != nil {
		return err
	}
	*claimed = item
	return nil
}

func (s *Store) completeItem(ctx context.Context, tx *firestorelib.Transaction, reference *firestorelib.DocumentRef, record itemRecord, item search.Item, offers []offer.Offer) error {
	job, err := s.readSearch(ctx, tx, item.SearchID)
	if err != nil {
		return err
	}
	item.LeaseOwner, item.LeaseUntil = "", nil
	record.Payload, record.AvailableAt = mustJSON(item), record.ExpiresAt.Add(doneParking)
	record.LeaseOwner, record.LeaseUntil = "", nil
	if err := tx.Set(reference, record); err != nil {
		return err
	}
	offerRecord := valueRecord{Payload: mustJSON(offers), ExpiresAt: record.ExpiresAt}
	if err := tx.Set(s.client.Collection("item_offers").Doc(item.ID), offerRecord); err != nil {
		return err
	}
	updateSearch(&job, item)
	return tx.Update(s.client.Collection("searches").Doc(job.ID), []firestorelib.Update{{Path: "payload", Value: mustJSON(job)}})
}

func (s *Store) loadItems(ctx context.Context, ids []string) ([]search.Item, error) {
	items := make([]search.Item, 0, len(ids))
	for _, id := range ids {
		document, err := s.client.Collection("items").Doc(id).Get(ctx)
		if err != nil {
			return nil, mapStoreError(err)
		}
		item, _, err := decodeItem(document)
		if err != nil {
			return nil, err
		}
		item.Offers = []offer.Offer{}
		if offers, err := s.client.Collection("item_offers").Doc(id).Get(ctx); err == nil {
			var record valueRecord
			_ = offers.DataTo(&record)
			_ = json.Unmarshal(record.Payload, &item.Offers)
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *Store) loadSource(ctx context.Context, domain string) (sourceRecord, error) {
	document, err := s.client.Collection("source_health").Doc(hashKey(domain)).Get(ctx)
	if status.Code(err) == codes.NotFound {
		return sourceRecord{Source: domain}, nil
	}
	if err != nil {
		return sourceRecord{}, err
	}
	var record sourceRecord
	err = document.DataTo(&record)
	return record, err
}

func (s *Store) readSource(ctx context.Context, tx *firestorelib.Transaction, reference *firestorelib.DocumentRef, domain string) (sourceRecord, error) {
	document, err := tx.Get(reference)
	if status.Code(err) == codes.NotFound {
		return sourceRecord{Source: domain}, nil
	}
	if err != nil {
		return sourceRecord{}, err
	}
	var record sourceRecord
	err = document.DataTo(&record)
	return record, err
}

func (s *Store) reserveTraffic(ctx context.Context, domain string) (time.Time, error) {
	reference := s.client.Collection("traffic_leases").Doc(hashKey(domain))
	var allowed time.Time
	err := s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		now := time.Now().UTC()
		allowed = now
		if document, err := tx.Get(reference); err == nil {
			if previous, err := document.DataAt("next_allowed_at"); err == nil && previous.(time.Time).After(now) {
				allowed = previous.(time.Time)
			}
		}
		return tx.Set(reference, map[string]any{"next_allowed_at": allowed.Add(250 * time.Millisecond)})
	})
	return allowed, err
}

func decodeSearch(document *firestorelib.DocumentSnapshot) (search.Job, error) {
	var record searchRecord
	if err := document.DataTo(&record); err != nil || record.ExpiresAt.Before(time.Now().UTC()) {
		return search.Job{}, search.ErrNotFound
	}
	var job search.Job
	err := json.Unmarshal(record.Payload, &job)
	return job, err
}

func decodeItem(document *firestorelib.DocumentSnapshot) (search.Item, itemRecord, error) {
	var record itemRecord
	if err := document.DataTo(&record); err != nil {
		return search.Item{}, record, err
	}
	var item search.Item
	err := json.Unmarshal(record.Payload, &item)
	item.LeaseOwner, item.LeaseUntil = record.LeaseOwner, record.LeaseUntil
	return item, record, err
}

func updateSearch(job *search.Job, item search.Item) {
	job.Processed++
	switch item.Status {
	case search.ItemFound:
		job.Found++
	case search.ItemNotFound:
		job.NotFound++
	case search.ItemSourceError:
		job.Errors++
	}
	if job.Processed == job.Total && job.Status != search.JobCancelled {
		now := time.Now().UTC()
		job.Status, job.CurrentCard, job.FinishedAt = search.JobCompleted, "", &now
		if job.Errors > 0 {
			job.Status = search.JobCompletedWithErrors
		}
	}
	job.UpdatedAt = time.Now().UTC()
}

func updateSource(record *sourceRecord, latency time.Duration, sourceErr error) {
	now := time.Now().UTC()
	record.LatencyMilliseconds = latency.Milliseconds()
	if sourceErr == nil {
		record.LastSuccess, record.ConsecutiveFailures, record.CircuitOpenUntil = &now, 0, nil
		return
	}
	record.LastFailure, record.ConsecutiveFailures = &now, record.ConsecutiveFailures+1
	if record.ConsecutiveFailures >= 5 {
		opened := now.Add(time.Minute)
		record.CircuitOpenUntil = &opened
	}
}

func mapSourceHealth(record sourceRecord) search.SourceHealth {
	return search.SourceHealth{
		Source: record.Source, LastSuccess: record.LastSuccess, LastFailure: record.LastFailure,
		ConsecutiveFailures: record.ConsecutiveFailures,
		LatencyMilliseconds: record.LatencyMilliseconds,
		CircuitOpenUntil:    record.CircuitOpenUntil,
	}
}

func waitUntil(ctx context.Context, target time.Time) error {
	wait := time.Until(target)
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func mapStoreError(err error) error {
	if status.Code(err) == codes.NotFound {
		return search.ErrNotFound
	}
	return err
}

func buildSearch(input search.CreateInput) search.Job {
	now := time.Now().UTC()
	return search.Job{ID: buildID("search"), Status: search.JobQueued, Total: len(input.Cards), CreatedAt: now, UpdatedAt: now}
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}

func hashKey(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func buildID(prefix string) string {
	data := make([]byte, 12)
	_, _ = rand.Read(data)
	return prefix + "_" + hex.EncodeToString(data)
}
