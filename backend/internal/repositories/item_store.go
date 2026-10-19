package repositories

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// One Turn discards this many dead Items before its final Claim. The Bound
// keeps a damaged Queue from holding one Function open without Limit.
const maxClaimDiscards = 20

// A finished Item is parked beyond its own Expiry. Parked exactly on it, it
// became claimable at the very Instant a Claim starts refusing it, and every
// Turn from then on wasted a Candidate Slot on a dead Document.
const doneParking = 24 * time.Hour

// ItemStore Claims, Renews and Completes the worker's card Items, and Counts
// the ones that Wait.
type ItemStore struct {
	database    Database
	collections Collections
	logger      *slog.Logger
}

// NewItemStore Reads the Queue through the Database's Collections.
func NewItemStore(database Database) *ItemStore {
	return &ItemStore{database: database, collections: database.Collections()}
}

// TellStore Gives the Store a Voice. Without one it stays silent, which is
// what every Test wants and what Production must never have.
func (l *ItemStore) TellStore(logger *slog.Logger) {
	l.logger = logger
}

func (l *ItemStore) say() *slog.Logger {
	if l.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return l.logger
}

func (l *ItemStore) ClaimSearchItem(ctx context.Context, owner string, lease time.Duration) (model.Item, error) {
	for attempt := 0; attempt <= maxClaimDiscards; attempt++ {
		claimed, discarded, err := l.claimOnce(ctx, owner, lease)
		if err != nil || claimed.ID != "" {
			return claimed, mapSearchError(err)
		}
		l.say().Warn("Claim Discarded Dead Item", "item", discarded)
	}
	return model.Item{}, search.ErrNotFound
}

// claimOnce Takes the oldest Item that is Ready, or Discards it when it cannot
// be Claimed: a Candidate that cannot be claimed is skipped, never fatal.
func (l *ItemStore) claimOnce(ctx context.Context, owner string, lease time.Duration) (model.Item, string, error) {
	var claimed model.Item
	var discarded string
	err := l.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		claimed, discarded = model.Item{}, ""
		candidates, err := tx.Items.List(ctx, Query{
			Filters: []Filter{{FieldAvailableAt, AtMost, time.Now().UTC()}},
			OrderBy: FieldAvailableAt,
			Limit:   1,
		})
		if err != nil {
			return err
		}
		if len(candidates) == 0 {
			return search.ErrNotFound
		}
		// claimItem refuses before it writes, so the Transaction stays clean
		// until one of them is actually taken.
		err = l.claimItem(ctx, tx, candidates[0], owner, lease, &claimed)
		if errors.Is(err, search.ErrNotFound) {
			discarded = candidates[0].ID
			return tx.Items.Delete(ctx, discarded)
		}
		return err
	})
	return claimed, discarded, err
}

func (l *ItemStore) claimItem(ctx context.Context, tx Collections, candidate Item, owner string, lease time.Duration, claimed *model.Item) error {
	if candidate.Damaged || lapsed(candidate.ExpiresAt) || candidate.SearchID == "" {
		return search.ErrNotFound
	}
	stored, err := readSearch(ctx, tx.Searches, candidate.SearchID)
	if errors.Is(err, search.ErrNotFound) || stored.Job.Status == model.JobCancelled {
		return search.ErrNotFound
	}
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	expires := now.Add(lease)
	candidate.Status, candidate.Attempts, candidate.LeaseOwner, candidate.LeaseUntil = model.ItemRunning, candidate.Attempts+1, owner, &expires
	candidate.AvailableAt = expires
	stored.Job.Status, stored.Job.CurrentCard, stored.Job.UpdatedAt = model.JobRunning, candidate.OriginalName, now
	if stored.Job.StartedAt == nil {
		stored.Job.StartedAt = &now
	}
	if err := tx.Items.Save(ctx, candidate.ID, candidate); err != nil {
		return err
	}
	if err := tx.Searches.Save(ctx, stored.Job.ID, stored); err != nil {
		return err
	}
	*claimed = candidate.Item
	return nil
}

// CountWaitingItems Says how many Items are ready for a Turn right now.
//
// A Task carries no Item: it means "wake up and take whatever is there", so
// the only Bond between Work and Wake-ups is the Count. A Turn that ends
// without working spends a Wake-up and nothing puts it back; the Sweeper
// compares this Count with the Queue and repairs the Difference.
func (l *ItemStore) CountWaitingItems(ctx context.Context, limit int) (int, error) {
	ready, err := l.collections.Items.List(ctx, Query{
		Filters: []Filter{{FieldAvailableAt, AtMost, time.Now().UTC()}},
		OrderBy: FieldAvailableAt,
		Limit:   limit,
	})
	if err != nil {
		return 0, mapSearchError(err)
	}
	waiting := 0
	for _, candidate := range ready {
		// A dead Document is not Work: counting it would ask for Turns that
		// only ever skip it.
		if !candidate.Damaged && !lapsed(candidate.ExpiresAt) {
			waiting++
		}
	}
	return waiting, nil
}

func (l *ItemStore) RenewItemLease(ctx context.Context, id, owner string, lease time.Duration) error {
	_, err := l.collections.Items.Update(ctx, id, func(stored Item) (Item, error) {
		if stored.Damaged || stored.LeaseOwner != owner || stored.Status != model.ItemRunning {
			return stored, search.ErrNotFound
		}
		expires := time.Now().UTC().Add(lease)
		stored.LeaseUntil, stored.AvailableAt = &expires, expires
		return stored, nil
	})
	return mapSearchError(err)
}

func (l *ItemStore) CompleteSearchItem(ctx context.Context, item model.Item, offers []model.Offer) error {
	err := l.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		stored, err := tx.Items.Get(ctx, item.ID)
		if err != nil {
			return err
		}
		if stored.Damaged || stored.LeaseOwner != item.LeaseOwner || stored.Status != model.ItemRunning {
			return search.ErrNotFound
		}
		parent, err := readSearch(ctx, tx.Searches, item.SearchID)
		if err != nil {
			return err
		}
		item.Sequence = parent.Job.Processed + 1
		item.LeaseOwner, item.LeaseUntil = "", nil
		stored.Item, stored.AvailableAt = item, stored.ExpiresAt.Add(doneParking)
		if err := tx.Items.Save(ctx, item.ID, stored); err != nil {
			return err
		}
		if err := tx.ItemOffers.Save(ctx, item.ID, Entry{Offers: offers, ExpiresAt: stored.ExpiresAt}); err != nil {
			return err
		}
		updateSearch(&parent.Job, item)
		return tx.Searches.Save(ctx, parent.Job.ID, parent)
	})
	return mapSearchError(err)
}

func updateSearch(job *model.Job, item model.Item) {
	job.Processed++
	switch item.Status {
	case model.ItemFound:
		job.Found++
	case model.ItemNotFound:
		job.NotFound++
	case model.ItemSourceError:
		job.Errors++
	}
	if job.Processed == job.Total && job.Status != model.JobCancelled {
		now := time.Now().UTC()
		job.Status, job.CurrentCard, job.FinishedAt = model.JobCompleted, "", &now
		if job.Errors > 0 {
			job.Status = model.JobCompletedWithErrors
		}
	}
	job.UpdatedAt = time.Now().UTC()
}
