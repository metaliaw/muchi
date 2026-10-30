package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// The Indexed Fields an Items Collection Keeps beside its Payload.
const (
	FieldSearchID    = "search_id"
	FieldSequence    = "completion_sequence"
	FieldAvailableAt = "available_at"
)

// SearchStore Opens, Reads and Cancels Searches on top of the Primitive. A
// retried Create or Cancel with the same Key and Payload Answers the Search it
// already Made, never a second one.
type SearchStore struct {
	database    Database
	collections Collections
	ttl         time.Duration
}

// NewSearchStore Keeps each Search and its Answers for ttl.
func NewSearchStore(database Database, ttl time.Duration) *SearchStore {
	return &SearchStore{database: database, collections: database.Collections(), ttl: ttl}
}

func (l *SearchStore) CreateSearch(ctx context.Context, key, hash string, input model.CreateInput) (model.Job, error) {
	var job model.Job
	var created bool
	err := l.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		created = false
		stored, found, err := readRequest(ctx, tx, "create", key)
		if err != nil {
			return err
		}
		if found {
			if stored.Hash != hash {
				return search.ErrConflict
			}
			answered, err := readSearch(ctx, tx.Searches, stored.ResourceID)
			job = answered.Job
			return err
		}
		job = buildSearch(input)
		created = true
		return l.saveSearch(ctx, tx, key, hash, job)
	})
	if err != nil || !created {
		return job, mapSearchError(err)
	}
	return job, l.createItems(ctx, job, input)
}

func (l *SearchStore) saveSearch(ctx context.Context, tx Collections, key, hash string, job model.Job) error {
	itemIDs := make([]string, job.Total)
	for index := range itemIDs {
		itemIDs[index] = buildID("item")
	}
	expires := time.Now().UTC().Add(l.ttl)
	if err := tx.Searches.Create(ctx, job.ID, Search{Job: job, ItemIDs: itemIDs, ExpiresAt: expires}); err != nil {
		return err
	}
	return saveRequest(ctx, tx, "create", key, Request{Hash: hash, ResourceID: job.ID, ExpiresAt: expires})
}

func (l *SearchStore) createItems(ctx context.Context, job model.Job, input model.CreateInput) error {
	stored, err := l.collections.Searches.Get(ctx, job.ID)
	if err != nil {
		return err
	}
	items := make([]Keyed[Item], len(input.Cards))
	for index, card := range input.Cards {
		item := model.Item{
			ID: stored.ItemIDs[index], SearchID: job.ID, Game: input.Game, Position: index,
			OriginalName: card.Name, NormalizedName: model.NormalizeCard(card.Name),
			Quantity: card.Quantity, Status: model.ItemPending, Offers: []model.Offer{},
			VerifyStock: input.Options.VerifyStock, Match: input.Options.Match,
			Kind: input.Options.Kind,
		}
		items[index] = Keyed[Item]{ID: item.ID, Value: Item{Item: item, AvailableAt: job.CreatedAt, ExpiresAt: stored.ExpiresAt}}
	}
	return l.collections.Items.SaveMany(ctx, items)
}

func (l *SearchStore) GetSearch(ctx context.Context, id string) (model.Job, error) {
	stored, err := readSearch(ctx, l.collections.Searches, id)
	return stored.Job, err
}

func (l *SearchStore) ListResults(ctx context.Context, id string, page model.ResultPage) (model.Result, error) {
	if _, err := readSearch(ctx, l.collections.Searches, id); err != nil {
		return model.Result{}, err
	}
	items, hasMore, err := l.loadResultPage(ctx, id, page)
	cursor := page.After
	if len(items) > 0 {
		cursor = items[len(items)-1].Sequence
	}
	return model.Result{SearchID: id, Items: items, Cursor: cursor, HasMore: hasMore}, err
}

// loadResultPage Reads one Page of finished Items, one more than the Page
// Holds so that the Last one Says whether another Page Follows.
func (l *SearchStore) loadResultPage(ctx context.Context, id string, page model.ResultPage) ([]model.Item, bool, error) {
	stored, err := l.collections.Items.List(ctx, Query{
		Filters: []Filter{{FieldSearchID, Equal, id}, {FieldSequence, Greater, page.After}},
		OrderBy: FieldSequence,
		Limit:   page.Limit + 1,
	})
	if err != nil {
		return nil, false, mapSearchError(err)
	}
	items := make([]model.Item, 0, page.Limit)
	for _, record := range stored {
		if record.Damaged {
			return nil, false, fmt.Errorf("item %s cannot be read", record.ID)
		}
		if len(items) == page.Limit {
			return items, true, nil
		}
		record.Offers = []model.Offer{}
		if offers, err := l.collections.ItemOffers.Get(ctx, record.ID); err == nil {
			record.Offers = offers.Offers
		}
		items = append(items, record.Item)
	}
	return items, false, nil
}

func (l *SearchStore) CancelSearch(ctx context.Context, id, key, hash string) (model.Job, error) {
	var job model.Job
	err := l.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		stored, found, err := readRequest(ctx, tx, "cancel", key)
		if err != nil {
			return err
		}
		if found {
			if stored.Hash != hash {
				return search.ErrConflict
			}
			answered, err := readSearch(ctx, tx.Searches, stored.ResourceID)
			job = answered.Job
			return err
		}
		cancelled, err := tx.Searches.Update(ctx, id, cancelSearch)
		if err != nil {
			return err
		}
		job = cancelled.Job
		return saveRequest(ctx, tx, "cancel", key, Request{Hash: hash, ResourceID: job.ID, ExpiresAt: job.UpdatedAt.Add(l.ttl)})
	})
	return job, mapSearchError(err)
}

// cancelSearch Closes a Search that is still Queued or Running, and Refuses any
// other.
func cancelSearch(stored Search) (Search, error) {
	if lapsed(stored.ExpiresAt) {
		return stored, search.ErrNotFound
	}
	if stored.Job.Status != model.JobQueued && stored.Job.Status != model.JobRunning {
		return stored, search.ErrNotRunning
	}
	now := time.Now().UTC()
	stored.Job.Status, stored.Job.FinishedAt, stored.Job.UpdatedAt = model.JobCancelled, &now, now
	return stored, nil
}

func buildSearch(input model.CreateInput) model.Job {
	now := time.Now().UTC()
	return model.Job{ID: buildID("search"), Game: input.Game, Status: model.JobQueued, Total: len(input.Cards), CreatedAt: now, UpdatedAt: now}
}
