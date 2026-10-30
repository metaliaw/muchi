package db

import (
	"context"
	"errors"

	firestorelib "cloud.google.com/go/firestore"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Collection Keeps one Kind of Record in one Firestore Collection. Pack Turns
// a Record into its Storage Shape and Unpack Reads it Back. Address Names the
// Document for an Id, and Nil Keeps the Id as it Comes. A Collection Bound to
// a Transaction Reads and Writes through it, and Leaves the Deadline to the
// Transaction.
type Collection[T any] struct {
	client  *firestorelib.Client
	name    string
	address func(string) string
	pack    func(T) any
	unpack  func(*firestorelib.DocumentSnapshot) (T, error)
	tx      *firestorelib.Transaction
}

var _ repositories.Repository[struct{}] = (*Collection[struct{}])(nil)

// Collections Opens every Collection the Stores Read through the Primitive.
func (s *Store) Collections() repositories.Collections {
	return s.collections(nil)
}

// Transact Runs work across Collections Bound to one Firestore Transaction.
func (s *Store) Transact(ctx context.Context, work func(ctx context.Context, tx repositories.Collections) error) error {
	ctx, cancel := boundContext(ctx)
	defer cancel()
	return s.client.RunTransaction(ctx, func(ctx context.Context, tx *firestorelib.Transaction) error {
		return work(ctx, s.collections(tx))
	})
}

func (s *Store) collections(tx *firestorelib.Transaction) repositories.Collections {
	return repositories.Collections{
		Searches:     &Collection[repositories.Search]{client: s.client, tx: tx, name: "searches", pack: packSearch, unpack: unpackSearch},
		Items:        &Collection[repositories.Item]{client: s.client, tx: tx, name: "items", pack: packItem, unpack: unpackItem},
		ItemOffers:   &Collection[repositories.Entry]{client: s.client, tx: tx, name: "item_offers", pack: packEntry, unpack: unpackEntry},
		Requests:     &Collection[repositories.Request]{client: s.client, tx: tx, name: "idempotency", address: hashKey, pack: packRequest, unpack: unpackRequest},
		Orders:       &Collection[model.Order]{client: s.client, tx: tx, name: "orders", pack: packOrder, unpack: decodeOrder},
		Offers:       &Collection[repositories.Snapshot]{client: s.client, tx: tx, name: "offers", address: hashKey, pack: packSnapshot, unpack: unpackSnapshot},
		OfferCache:   &Collection[repositories.Entry]{client: s.client, tx: tx, name: "offer_cache", address: hashKey, pack: packEntry, unpack: unpackEntry},
		SourceHealth: &Collection[model.SourceHealth]{client: s.client, tx: tx, name: "source_health", address: hashKey, pack: packSourceHealth, unpack: decodeSourceHealth},
		Traffic:      &Collection[repositories.Lease]{client: s.client, tx: tx, name: "traffic_leases", address: hashKey, pack: packLease, unpack: unpackLease},
		Health:       &Collection[repositories.Ping]{client: s.client, tx: tx, name: "health", pack: packPing, unpack: unpackPing},
	}
}

func (c *Collection[T]) document(id string) *firestorelib.DocumentRef {
	if c.address != nil {
		id = c.address(id)
	}
	return c.client.Collection(c.name).Doc(id)
}

// bound Shortens the Deadline unless a Transaction already Owns it.
func (c *Collection[T]) bound(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.tx != nil {
		return ctx, func() {}
	}
	return boundContext(ctx)
}

func (c *Collection[T]) Create(ctx context.Context, id string, value T) error {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	if c.tx != nil {
		return mapCollectionError(c.tx.Create(c.document(id), c.pack(value)))
	}
	_, err := c.document(id).Create(ctx, c.pack(value))
	return mapCollectionError(err)
}

func (c *Collection[T]) Save(ctx context.Context, id string, value T) error {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	if c.tx != nil {
		return c.tx.Set(c.document(id), c.pack(value))
	}
	_, err := c.document(id).Set(ctx, c.pack(value))
	return err
}

func (c *Collection[T]) Get(ctx context.Context, id string) (T, error) {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	document, err := c.read(ctx, c.document(id))
	if err != nil {
		var none T
		return none, mapCollectionError(err)
	}
	return c.unpack(document)
}

func (c *Collection[T]) read(ctx context.Context, reference *firestorelib.DocumentRef) (*firestorelib.DocumentSnapshot, error) {
	if c.tx != nil {
		return c.tx.Get(reference)
	}
	return reference.Get(ctx)
}

func (c *Collection[T]) Update(ctx context.Context, id string, change func(T) (T, error)) (T, error) {
	var updated T
	reference := c.document(id)
	modify := func(ctx context.Context, tx *firestorelib.Transaction) error {
		document, err := tx.Get(reference)
		if err != nil {
			return mapCollectionError(err)
		}
		current, err := c.unpack(document)
		if err != nil {
			return err
		}
		updated, err = change(current)
		if err != nil {
			return err
		}
		return tx.Set(reference, c.pack(updated))
	}
	var err error
	if c.tx != nil {
		err = modify(ctx, c.tx)
	} else {
		bounded, cancel := boundContext(ctx)
		defer cancel()
		err = c.client.RunTransaction(bounded, modify)
	}
	if err != nil {
		var none T
		return none, err
	}
	return updated, nil
}

func (c *Collection[T]) Delete(ctx context.Context, id string) error {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	if c.tx != nil {
		return c.tx.Delete(c.document(id))
	}
	_, err := c.document(id).Delete(ctx)
	return err
}

func (c *Collection[T]) List(ctx context.Context, query repositories.Query) ([]T, error) {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	built := c.buildQuery(query)
	documents := built.Documents(ctx)
	if c.tx != nil {
		documents = c.tx.Documents(built)
	}
	defer documents.Stop()
	records := make([]T, 0)
	for {
		document, err := documents.Next()
		if errors.Is(err, iterator.Done) {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		record, err := c.unpack(document)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
}

func (c *Collection[T]) buildQuery(query repositories.Query) firestorelib.Query {
	built := c.client.Collection(c.name).Query
	for _, filter := range query.Filters {
		built = built.Where(filter.Field, string(filter.Operator), filter.Value)
	}
	if query.OrderBy != "" {
		built = built.OrderBy(query.OrderBy, firestorelib.Asc)
	}
	if query.Limit > 0 {
		built = built.Limit(query.Limit)
	}
	return built
}

// GetMany Reads the Ids in Groups no Larger than one Batch.
func (c *Collection[T]) GetMany(ctx context.Context, ids []string) ([]repositories.Keyed[T], error) {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	found := make([]repositories.Keyed[T], 0, len(ids))
	for start := 0; start < len(ids); start += maxBatchWrites {
		group := ids[start:min(start+maxBatchWrites, len(ids))]
		references := make([]*firestorelib.DocumentRef, len(group))
		for index, id := range group {
			references[index] = c.document(id)
		}
		documents, err := c.readAll(ctx, references)
		if err != nil {
			return nil, err
		}
		for index, document := range documents {
			if !document.Exists() {
				continue
			}
			record, err := c.unpack(document)
			if err != nil {
				return nil, err
			}
			found = append(found, repositories.Keyed[T]{ID: group[index], Value: record})
		}
	}
	return found, nil
}

func (c *Collection[T]) readAll(ctx context.Context, references []*firestorelib.DocumentRef) ([]*firestorelib.DocumentSnapshot, error) {
	if c.tx != nil {
		return c.tx.GetAll(references)
	}
	return c.client.GetAll(ctx, references)
}

// SaveMany Writes the Records in Batches no Larger than Firestore Accepts.
func (c *Collection[T]) SaveMany(ctx context.Context, records []repositories.Keyed[T]) error {
	ctx, cancel := c.bound(ctx)
	defer cancel()
	if c.tx != nil {
		for _, record := range records {
			if err := c.tx.Set(c.document(record.ID), c.pack(record.Value)); err != nil {
				return err
			}
		}
		return nil
	}
	for start := 0; start < len(records); start += maxBatchWrites {
		batch := c.client.Batch()
		for _, record := range records[start:min(start+maxBatchWrites, len(records))] {
			batch.Set(c.document(record.ID), c.pack(record.Value))
		}
		if _, err := batch.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func mapCollectionError(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return repositories.ErrMissing
	case codes.AlreadyExists:
		return repositories.ErrExists
	}
	return err
}
