package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/source"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

type jobRecord struct {
	ID, Status, CurrentCard, IncidentID string
	Total, Processed, Found             int
	NotFound, Errors                    int
	CreatedAt, UpdatedAt                time.Time
	StartedAt, FinishedAt               *time.Time
}

type itemRecord struct {
	ID, SearchID, OriginalName, NormalizedName string
	Status, Source, ErrorCode, ErrorMessage    string
	Position, Quantity, Attempts               int
	LeaseOwner                                 string
	LeaseUntil                                 *time.Time
	VerifyStock, StoresOnly                    bool
}

type offerRecord struct {
	ID, SearchItemID, CardName, Store      string
	PriceAmount, PriceCurrency, URL        string
	VariantID, Language, Condition, Finish string
	Source, StockStatus, SuspiciousReason  string
	Suspicious                             bool
	Metadata                               []byte
}

type idempotencyRecord struct {
	Key, Action, RequestHash, ResourceID string
}

type cacheRecord struct {
	CacheKey, Kind string
	Payload        []byte
	ExpiresAt      time.Time
}

func OpenStore(dsn string) (*Store, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, fmt.Errorf("open search store: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) CreateSearch(ctx context.Context, key, hash string, input search.CreateInput) (search.Job, error) {
	var result search.Job
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, found, err := loadIdempotent(tx, key, "create_search", hash)
		if err != nil || found {
			result = job
			return err
		}
		result, err = insertSearch(tx, key, hash, input)
		return err
	})
	return result, mapStoreError(err)
}

func (s *Store) GetSearch(ctx context.Context, id string) (search.Job, error) {
	var record jobRecord
	err := s.db.WithContext(ctx).Table("search_jobs").Where("id = ?", id).Take(&record).Error
	return mapJob(record), mapStoreError(err)
}

func (s *Store) ListResults(ctx context.Context, id string) (search.Result, error) {
	if _, err := s.GetSearch(ctx, id); err != nil {
		return search.Result{}, err
	}
	var records []itemRecord
	err := s.db.WithContext(ctx).Table("search_items").Where("search_id = ?", id).Order("position").Find(&records).Error
	if err != nil {
		return search.Result{}, mapStoreError(err)
	}
	items := make([]search.Item, 0, len(records))
	for _, record := range records {
		item := mapItem(record)
		item.Offers, err = s.loadOffers(ctx, record.ID)
		if err != nil {
			return search.Result{}, err
		}
		items = append(items, item)
	}
	return search.Result{SearchID: id, Items: items}, nil
}

func (s *Store) CancelSearch(ctx context.Context, id, key, hash string) (search.Job, error) {
	var result search.Job
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		job, found, err := loadIdempotent(tx, key, "cancel_search", hash)
		if err != nil || found {
			result = job
			return err
		}
		return cancelSearch(tx, id, key, hash, &result)
	})
	return result, mapStoreError(err)
}

func (s *Store) ClaimSearchItem(ctx context.Context, owner string, lease time.Duration) (search.Item, error) {
	var record itemRecord
	expires := time.Now().UTC().Add(lease)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := `SELECT i.* FROM search_items i JOIN search_jobs j ON j.id = i.search_id
			WHERE j.status IN ('queued','running') AND (i.status = 'pending' OR (i.status = 'running' AND i.lease_until < NOW()))
			ORDER BY j.created_at, i.position FOR UPDATE OF i SKIP LOCKED LIMIT 1`
		if err := tx.Raw(query).Scan(&record).Error; err != nil {
			return err
		}
		if record.ID == "" {
			return search.ErrNotFound
		}
		return markClaimed(tx, &record, owner, expires)
	})
	return mapItem(record), mapStoreError(err)
}

func (s *Store) RenewItemLease(ctx context.Context, id, owner string, lease time.Duration) error {
	expires := time.Now().UTC().Add(lease)
	result := s.db.WithContext(ctx).Table("search_items").Where("id = ? AND lease_owner = ? AND status = 'running'", id, owner).Update("lease_until", expires)
	if result.Error != nil {
		return mapStoreError(result.Error)
	}
	if result.RowsAffected == 0 {
		return search.ErrNotFound
	}
	return nil
}

func (s *Store) CompleteSearchItem(ctx context.Context, item search.Item, offers []offer.Offer) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Table("search_items").Where("id = ? AND lease_owner = ? AND status = 'running'", item.ID, item.LeaseOwner).Update("updated_at", time.Now().UTC())
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return search.ErrNotFound
		}
		if err := tx.Exec("DELETE FROM offers WHERE search_item_id = ?", item.ID).Error; err != nil {
			return err
		}
		if err := insertOffers(tx, item.ID, offers); err != nil {
			return err
		}
		return finishItem(tx, item)
	})
	return mapStoreError(err)
}

func (s *Store) LoadOffers(ctx context.Context, key string) ([]offer.Offer, bool, error) {
	var record cacheRecord
	err := s.db.WithContext(ctx).Table("offer_cache").Where("cache_key = ? AND expires_at > NOW()", key).Take(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var items []offer.Offer
	if err := json.Unmarshal(record.Payload, &items); err != nil {
		return nil, false, err
	}
	return items, true, nil
}

func (s *Store) SaveOffers(ctx context.Context, key string, items []offer.Offer, ttl time.Duration) error {
	payload, err := json.Marshal(items)
	if err != nil {
		return err
	}
	record := cacheRecord{CacheKey: key, Kind: cacheKind(items), Payload: payload, ExpiresAt: time.Now().UTC().Add(ttl)}
	conflict := clause.OnConflict{Columns: []clause.Column{{Name: "cache_key"}}, DoUpdates: clause.AssignmentColumns([]string{"kind", "payload", "expires_at", "updated_at"})}
	return s.db.WithContext(ctx).Table("offer_cache").Clauses(conflict).Create(&record).Error
}

func (s *Store) AwaitSource(ctx context.Context, domain string) error {
	var health struct {
		CircuitOpenUntil *time.Time
	}
	if err := s.db.WithContext(ctx).Table("source_health").Select("circuit_open_until").Where("source = ?", domain).Scan(&health).Error; err != nil {
		return err
	}
	if health.CircuitOpenUntil != nil && health.CircuitOpenUntil.After(time.Now().UTC()) {
		return fmt.Errorf("%w until %s", source.ErrCircuitOpen, health.CircuitOpenUntil.Format(time.RFC3339))
	}
	query := `INSERT INTO traffic_leases (domain, next_allowed_at, updated_at) VALUES (?, NOW(), NOW())
		ON CONFLICT (domain) DO UPDATE SET next_allowed_at = GREATEST(traffic_leases.next_allowed_at, NOW()) + INTERVAL '250 milliseconds', updated_at = NOW()
		RETURNING next_allowed_at`
	var allowed time.Time
	if err := s.db.WithContext(ctx).Raw(query, domain).Scan(&allowed).Error; err != nil {
		return err
	}
	wait := time.Until(allowed)
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

func (s *Store) RecordSource(ctx context.Context, domain string, latency time.Duration, sourceErr error) error {
	succeeded := sourceErr == nil
	query := `INSERT INTO source_health (source, last_success, last_failure, consecutive_failures, latency_ms, updated_at)
		VALUES (?, CASE WHEN ? THEN NOW() END, CASE WHEN ? THEN NULL ELSE NOW() END, CASE WHEN ? THEN 0 ELSE 1 END, ?, NOW())
		ON CONFLICT (source) DO UPDATE SET last_success = CASE WHEN ? THEN NOW() ELSE source_health.last_success END,
		last_failure = CASE WHEN ? THEN source_health.last_failure ELSE NOW() END,
		consecutive_failures = CASE WHEN ? THEN 0 ELSE source_health.consecutive_failures + 1 END,
		latency_ms = EXCLUDED.latency_ms, circuit_open_until = CASE WHEN NOT ? AND source_health.consecutive_failures >= 4 THEN NOW() + INTERVAL '1 minute' ELSE NULL END, updated_at = NOW()`
	return s.db.WithContext(ctx).Exec(query, domain, succeeded, succeeded, succeeded, latency.Milliseconds(), succeeded, succeeded, succeeded, succeeded).Error
}

func (s *Store) CheckHealth(ctx context.Context) error {
	db, err := s.db.DB()
	if err != nil {
		return err
	}
	return db.PingContext(ctx)
}

func (s *Store) ListSourceHealth(ctx context.Context) ([]search.SourceHealth, error) {
	var records []struct {
		Source              string
		LastSuccess         *time.Time
		LastFailure         *time.Time
		ConsecutiveFailures int
		LatencyMS           int64 `gorm:"column:latency_ms"`
		CircuitOpenUntil    *time.Time
	}
	if err := s.db.WithContext(ctx).Table("source_health").Order("source").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]search.SourceHealth, 0, len(records))
	for _, record := range records {
		result = append(result, search.SourceHealth{Source: record.Source, LastSuccess: record.LastSuccess, LastFailure: record.LastFailure, ConsecutiveFailures: record.ConsecutiveFailures, LatencyMilliseconds: record.LatencyMS, CircuitOpenUntil: record.CircuitOpenUntil})
	}
	return result, nil
}

func insertSearch(tx *gorm.DB, key, hash string, input search.CreateInput) (search.Job, error) {
	now := time.Now().UTC()
	record := jobRecord{ID: makeID("search"), Status: string(search.JobQueued), Total: len(input.Cards), CreatedAt: now, UpdatedAt: now}
	if err := tx.Table("search_jobs").Create(&record).Error; err != nil {
		return search.Job{}, err
	}
	for position, card := range input.Cards {
		item := itemRecord{ID: makeID("item"), SearchID: record.ID, Position: position, OriginalName: card.Name, NormalizedName: offer.NormalizeCard(card.Name), Quantity: card.Quantity, Status: string(search.ItemPending), VerifyStock: input.Options.VerifyStock, StoresOnly: input.Options.StoresOnly}
		if err := tx.Table("search_items").Create(&item).Error; err != nil {
			return search.Job{}, err
		}
	}
	idem := idempotencyRecord{Key: key, Action: "create_search", RequestHash: hash, ResourceID: record.ID}
	return mapJob(record), tx.Table("idempotency_keys").Create(&idem).Error
}

func loadIdempotent(tx *gorm.DB, key, action, hash string) (search.Job, bool, error) {
	var idem idempotencyRecord
	err := tx.Table("idempotency_keys").Where("key = ? AND action = ?", key, action).Take(&idem).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return search.Job{}, false, nil
	}
	if err != nil {
		return search.Job{}, false, err
	}
	if idem.RequestHash != hash {
		return search.Job{}, true, search.ErrConflict
	}
	var record jobRecord
	err = tx.Table("search_jobs").Where("id = ?", idem.ResourceID).Take(&record).Error
	return mapJob(record), true, err
}

func cancelSearch(tx *gorm.DB, id, key, hash string, result *search.Job) error {
	var record jobRecord
	if err := tx.Table("search_jobs").Where("id = ?", id).Clauses(clause.Locking{Strength: "UPDATE"}).Take(&record).Error; err != nil {
		return err
	}
	if record.Status != string(search.JobQueued) && record.Status != string(search.JobRunning) {
		return search.ErrNotRunning
	}
	now := time.Now().UTC()
	if err := tx.Table("search_jobs").Where("id = ?", id).Updates(map[string]any{"status": search.JobCancelled, "finished_at": now, "updated_at": now}).Error; err != nil {
		return err
	}
	idem := idempotencyRecord{Key: key, Action: "cancel_search", RequestHash: hash, ResourceID: id}
	if err := tx.Table("idempotency_keys").Create(&idem).Error; err != nil {
		return err
	}
	record.Status, record.FinishedAt, record.UpdatedAt = string(search.JobCancelled), &now, now
	*result = mapJob(record)
	return nil
}

func markClaimed(tx *gorm.DB, record *itemRecord, owner string, expires time.Time) error {
	now := time.Now().UTC()
	updates := map[string]any{"status": search.ItemRunning, "attempts": gorm.Expr("attempts + 1"), "lease_owner": owner, "lease_until": expires, "started_at": gorm.Expr("COALESCE(started_at, ?)", now), "updated_at": now}
	if err := tx.Table("search_items").Where("id = ?", record.ID).Updates(updates).Error; err != nil {
		return err
	}
	job := map[string]any{"status": search.JobRunning, "current_card": record.OriginalName, "started_at": gorm.Expr("COALESCE(started_at, ?)", now), "updated_at": now}
	if err := tx.Table("search_jobs").Where("id = ?", record.SearchID).Updates(job).Error; err != nil {
		return err
	}
	record.Status, record.Attempts, record.LeaseOwner, record.LeaseUntil = string(search.ItemRunning), record.Attempts+1, owner, &expires
	return nil
}

func insertOffers(tx *gorm.DB, itemID string, items []offer.Offer) error {
	for _, item := range items {
		metadata, _ := json.Marshal(item.Metadata)
		record := offerRecord{ID: makeID("offer"), SearchItemID: itemID, CardName: item.CardName, Store: item.Store, PriceAmount: item.PriceAmount, PriceCurrency: item.PriceCurrency, URL: item.URL, VariantID: item.VariantID, Language: item.Language, Condition: item.Condition, Finish: item.Finish, Source: item.Source, StockStatus: item.StockStatus, Suspicious: item.Suspicious, SuspiciousReason: item.SuspiciousReason, Metadata: metadata}
		if err := tx.Table("offers").Create(&record).Error; err != nil {
			return err
		}
	}
	return nil
}

func finishItem(tx *gorm.DB, item search.Item) error {
	now := time.Now().UTC()
	updates := map[string]any{"status": item.Status, "source": item.Source, "error_code": item.ErrorCode, "error_message": item.ErrorMessage, "lease_owner": "", "lease_until": nil, "finished_at": now, "updated_at": now}
	if err := tx.Table("search_items").Where("id = ?", item.ID).Updates(updates).Error; err != nil {
		return err
	}
	query := `UPDATE search_jobs j SET processed = s.processed, found = s.found, not_found = s.not_found, errors = s.errors,
		status = CASE WHEN j.status = 'cancelled' THEN j.status WHEN s.processed = j.total AND s.errors > 0 THEN 'completed_with_errors' WHEN s.processed = j.total THEN 'completed' ELSE 'running' END,
		current_card = CASE WHEN s.processed = j.total THEN '' ELSE j.current_card END,
		finished_at = CASE WHEN s.processed = j.total THEN NOW() ELSE NULL END, updated_at = NOW()
		FROM (SELECT search_id, COUNT(*) FILTER (WHERE status IN ('found','not_found','source_error')) processed,
		COUNT(*) FILTER (WHERE status = 'found') found, COUNT(*) FILTER (WHERE status = 'not_found') not_found,
		COUNT(*) FILTER (WHERE status = 'source_error') errors FROM search_items WHERE search_id = ? GROUP BY search_id) s
		WHERE j.id = s.search_id`
	return tx.Exec(query, item.SearchID).Error
}

func (s *Store) loadOffers(ctx context.Context, itemID string) ([]offer.Offer, error) {
	var records []offerRecord
	if err := s.db.WithContext(ctx).Table("offers").Where("search_item_id = ?", itemID).Order("price_amount").Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]offer.Offer, 0, len(records))
	for _, record := range records {
		var metadata map[string]string
		_ = json.Unmarshal(record.Metadata, &metadata)
		result = append(result, offer.Offer{ID: record.ID, CardName: record.CardName, Store: record.Store, PriceAmount: record.PriceAmount, PriceCurrency: record.PriceCurrency, URL: record.URL, VariantID: record.VariantID, Language: record.Language, Condition: record.Condition, Finish: record.Finish, Source: record.Source, StockStatus: record.StockStatus, Suspicious: record.Suspicious, SuspiciousReason: record.SuspiciousReason, Metadata: metadata})
	}
	return result, nil
}

func mapJob(record jobRecord) search.Job {
	return search.Job{ID: record.ID, Status: search.JobStatus(record.Status), Total: record.Total, Processed: record.Processed, CurrentCard: record.CurrentCard, Found: record.Found, NotFound: record.NotFound, Errors: record.Errors, CreatedAt: record.CreatedAt, StartedAt: record.StartedAt, UpdatedAt: record.UpdatedAt, FinishedAt: record.FinishedAt, IncidentID: record.IncidentID}
}

func mapItem(record itemRecord) search.Item {
	return search.Item{ID: record.ID, SearchID: record.SearchID, Position: record.Position, OriginalName: record.OriginalName, NormalizedName: record.NormalizedName, Quantity: record.Quantity, Status: search.ItemStatus(record.Status), Attempts: record.Attempts, Source: record.Source, ErrorCode: record.ErrorCode, ErrorMessage: record.ErrorMessage, LeaseOwner: record.LeaseOwner, LeaseUntil: record.LeaseUntil, VerifyStock: record.VerifyStock, StoresOnly: record.StoresOnly, Offers: []offer.Offer{}}
}

func mapStoreError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return search.ErrNotFound
	}
	return err
}

func makeID(prefix string) string {
	data := make([]byte, 12)
	_, _ = rand.Read(data)
	return prefix + "_" + hex.EncodeToString(data)
}

func cacheKind(items []offer.Offer) string {
	if len(items) == 0 {
		return "negative"
	}
	return "positive"
}
