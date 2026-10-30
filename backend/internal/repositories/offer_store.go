package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// OfferStore Caches Offers under a Key and Reads each one back by its own Id
// through a Snapshot that Lives as long as the Entry that Carried it.
type OfferStore struct {
	offers Repository[Snapshot]
	cache  Repository[Entry]
}

// NewOfferStore Reads the Entries and Snapshots through the Database's Collections.
func NewOfferStore(database Database) *OfferStore {
	collections := database.Collections()
	return &OfferStore{offers: collections.Offers, cache: collections.OfferCache}
}

func (s *OfferStore) LoadOffers(ctx context.Context, key string) ([]model.Offer, bool, error) {
	entry, err := s.cache.Get(ctx, key)
	if errors.Is(err, ErrMissing) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if lapsed(entry.ExpiresAt) {
		return nil, false, nil
	}
	if !entry.Indexed {
		if err := s.indexMissingOffers(ctx, entry.Offers, entry.ExpiresAt); err != nil {
			return nil, false, err
		}
		if err := s.markIndexed(ctx, key); err != nil {
			return nil, false, err
		}
	}
	return entry.Offers, true, nil
}

func (s *OfferStore) SaveOffers(ctx context.Context, key string, offers []model.Offer, ttl time.Duration) error {
	expiresAt := time.Now().UTC().Add(ttl)
	if err := s.cache.Save(ctx, key, Entry{Offers: offers, ExpiresAt: expiresAt}); err != nil {
		return err
	}
	if err := s.offers.SaveMany(ctx, snapshotsOf(offers, expiresAt)); err != nil {
		return err
	}
	return s.markIndexed(ctx, key)
}

// markIndexed Records that every Offer of an Entry has its own Snapshot.
func (s *OfferStore) markIndexed(ctx context.Context, key string) error {
	_, err := s.cache.Update(ctx, key, func(entry Entry) (Entry, error) {
		entry.Indexed = true
		return entry, nil
	})
	return err
}

// indexMissingOffers Writes a Snapshot for each Offer that has none Alive,
// and Leaves the Alive ones as they Are.
func (s *OfferStore) indexMissingOffers(ctx context.Context, offers []model.Offer, expiresAt time.Time) error {
	snapshots := snapshotsOf(offers, expiresAt)
	ids := make([]string, len(snapshots))
	for index, snapshot := range snapshots {
		ids[index] = snapshot.ID
	}
	kept, err := s.offers.GetMany(ctx, ids)
	if err != nil {
		return err
	}
	alive := make(map[string]bool, len(kept))
	for _, record := range kept {
		alive[record.ID] = record.Value.ExpiresAt.After(time.Now().UTC())
	}
	missing := make([]Keyed[Snapshot], 0, len(snapshots))
	for _, snapshot := range snapshots {
		if !alive[snapshot.ID] {
			missing = append(missing, snapshot)
		}
	}
	return s.offers.SaveMany(ctx, missing)
}

// snapshotsOf Gives each Offer that has an Id its own Snapshot.
func snapshotsOf(offers []model.Offer, expiresAt time.Time) []Keyed[Snapshot] {
	snapshots := make([]Keyed[Snapshot], 0, len(offers))
	for _, offer := range offers {
		if offer.ID != "" {
			snapshots = append(snapshots, Keyed[Snapshot]{ID: offer.ID, Value: Snapshot{Offer: offer, ExpiresAt: expiresAt}})
		}
	}
	return snapshots
}

// GetOffer Reads one Stable Offer Snapshot until the Cache Entry Expires. A
// Snapshot that Answers to another Id is Missing, never Borrowed.
func (s *OfferStore) GetOffer(ctx context.Context, id string) (model.Offer, error) {
	snapshot, err := s.offers.Get(ctx, id)
	if errors.Is(err, ErrMissing) || (err == nil && !snapshot.ExpiresAt.After(time.Now().UTC())) {
		return model.Offer{}, search.ErrNotFound
	}
	if err != nil {
		return model.Offer{}, err
	}
	if snapshot.Offer.ID != id {
		return model.Offer{}, search.ErrNotFound
	}
	return snapshot.Offer, nil
}

// DropOffers Forgets one Cached Entry; a Missing Entry is already Forgotten.
func (s *OfferStore) DropOffers(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}
