package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// The Indexed Fields an Orders Collection Keeps beside its Payload, and the
// only ones the Store Filters on. `status` with `updated_at` Needs a
// Composite Index, declared in firestore.indexes.json.
const (
	FieldStatus     = "status"
	FieldDomain     = "domain"
	FieldStoreOrder = "store_order"
	FieldUpdatedAt  = "updated_at"
)

// OrderStore Places Orders and Moves them between Statuses on top of the
// Orders Collection. Every Move Repeats its from-Status Guard inside the
// Update, so a Caller that Lost the Race gets a Conflict, never an Overwrite.
type OrderStore struct {
	database Database
	orders   Repository[model.Order]
	ttl      time.Duration
}

// NewOrderStore Keeps each Placed Order's Answer for ttl.
func NewOrderStore(database Database, ttl time.Duration) *OrderStore {
	return &OrderStore{database: database, orders: database.Collections().Orders, ttl: ttl}
}

// CreateOrder Reserves the Stock a Store's own Checkout just Committed. The
// Caller Names the Status, since a Store that Redirects to Payment and one
// that Waits for a Transfer both Start Pending, and nothing here Decides
// which.
//
// A retried Call with the same Key and Payload Answers the Order it already
// Placed, never a second one: a Buyer's double Tap must not become two
// Orders at the same Store.
func (l *OrderStore) CreateOrder(ctx context.Context, key, hash string, order model.Order) (model.Order, error) {
	var placed model.Order
	err := l.database.Transact(ctx, func(ctx context.Context, tx Collections) error {
		stored, found, err := readRequest(ctx, tx, "place-order", key)
		if err != nil {
			return err
		}
		if found {
			if stored.Hash != hash {
				return search.ErrConflict
			}
			placed, err = tx.Orders.Get(ctx, stored.ResourceID)
			return mapOrderError(err)
		}
		now := time.Now().UTC()
		order.ID = buildID("order")
		order.CreatedAt, order.UpdatedAt = now, now
		placed = order
		if err := tx.Orders.Create(ctx, order.ID, order); err != nil {
			return err
		}
		return saveRequest(ctx, tx, "place-order", key, Request{Hash: hash, ResourceID: order.ID, ExpiresAt: now.Add(l.ttl)})
	})
	if err != nil {
		return model.Order{}, mapSearchError(err)
	}
	return placed, nil
}

// GetOrder Reads one Order back by its Id.
func (l *OrderStore) GetOrder(ctx context.Context, id string) (model.Order, error) {
	order, err := l.orders.Get(ctx, id)
	return order, mapOrderError(err)
}

// FindOrderByStoreOrder Reads the Order a Store's own Webhook Names by its
// own Order Id, the only Id a Store Payload Carries, never Muchi's.
func (l *OrderStore) FindOrderByStoreOrder(ctx context.Context, domain, storeOrder string) (model.Order, error) {
	found, err := l.orders.List(ctx, Query{
		Filters: []Filter{{FieldDomain, Equal, domain}, {FieldStoreOrder, Equal, storeOrder}},
		Limit:   1,
	})
	if err != nil {
		return model.Order{}, err
	}
	if len(found) == 0 {
		return model.Order{}, model.ErrOrderNotFound
	}
	return found[0], nil
}

// MoveOrderStatus Advances an Order from one Status to the next, never past
// where the Caller Believes it Stands: a Webhook Confirming an Order the
// Sweeper already Released must Lose, not Overwrite the Release.
func (l *OrderStore) MoveOrderStatus(ctx context.Context, id string, from, to model.OrderStatus) (model.Order, error) {
	order, err := l.orders.Update(ctx, id, func(order model.Order) (model.Order, error) {
		if order.Status != from {
			return order, model.ErrOrderConflict
		}
		order.Status, order.UpdatedAt = to, time.Now().UTC()
		return order, nil
	})
	return order, mapOrderError(err)
}

// ReportOrder Moves a `linked` Order to `reported` with the Number the Buyer
// Read off the Store: the Status and its Number Land together or not at all.
func (l *OrderStore) ReportOrder(ctx context.Context, id, storeOrder string) (model.Order, error) {
	order, err := l.orders.Update(ctx, id, func(order model.Order) (model.Order, error) {
		if order.Status != model.OrderLinked {
			return order, model.ErrOrderConflict
		}
		order.Status, order.StoreOrder, order.UpdatedAt = model.OrderReported, storeOrder, time.Now().UTC()
		return order, nil
	})
	return order, mapOrderError(err)
}

// ReleaseExpiredOrders Moves every `pending` Order whose own Cutoff already
// Passed to `released`, and Answers how many it Moved. A Confirm that Landed
// between the List and the Move must Win, so a Conflict is skipped, never fatal.
func (l *OrderStore) ReleaseExpiredOrders(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	stale, err := l.orders.List(ctx, Query{
		Filters: []Filter{{FieldStatus, Equal, string(model.OrderPending)}, {FieldUpdatedAt, AtMost, cutoff}},
		OrderBy: FieldUpdatedAt,
		Limit:   limit,
	})
	if err != nil {
		return 0, err
	}
	released := 0
	for _, order := range stale {
		_, err := l.MoveOrderStatus(ctx, order.ID, model.OrderPending, model.OrderReleased)
		if errors.Is(err, model.ErrOrderConflict) {
			continue
		}
		if err != nil {
			return released, err
		}
		released++
	}
	return released, nil
}

func mapOrderError(err error) error {
	if errors.Is(err, ErrMissing) {
		return model.ErrOrderNotFound
	}
	return err
}
