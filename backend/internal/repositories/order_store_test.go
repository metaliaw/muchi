package repositories_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
)

// shelf Keeps Orders in Memory and Answers the only Queries the Store Asks.
type shelf struct{ kept map[string]model.Order }

func (s *shelf) Create(_ context.Context, id string, order model.Order) error {
	s.kept[id] = order
	return nil
}

func (s *shelf) Get(_ context.Context, id string) (model.Order, error) {
	order, found := s.kept[id]
	if !found {
		return model.Order{}, repositories.ErrMissing
	}
	return order, nil
}

func (s *shelf) Update(_ context.Context, id string, change func(model.Order) (model.Order, error)) (model.Order, error) {
	order, err := s.Get(context.Background(), id)
	if err != nil {
		return model.Order{}, err
	}
	changed, err := change(order)
	if err != nil {
		return model.Order{}, err
	}
	s.kept[id] = changed
	return changed, nil
}

func (s *shelf) Save(_ context.Context, id string, order model.Order) error {
	s.kept[id] = order
	return nil
}

func (s *shelf) GetMany(context.Context, []string) ([]repositories.Keyed[model.Order], error) {
	return nil, errors.New("the shelf keeps no bulk reads")
}

func (s *shelf) SaveMany(context.Context, []repositories.Keyed[model.Order]) error {
	return errors.New("the shelf keeps no bulk writes")
}

func (s *shelf) Delete(_ context.Context, id string) error {
	delete(s.kept, id)
	return nil
}

func (s *shelf) List(_ context.Context, query repositories.Query) ([]model.Order, error) {
	listed := make([]model.Order, 0)
	for _, order := range s.kept {
		if matches(order, query.Filters) {
			listed = append(listed, order)
		}
	}
	return listed, nil
}

func matches(order model.Order, filters []repositories.Filter) bool {
	for _, filter := range filters {
		switch filter.Field {
		case repositories.FieldDomain:
			if order.Domain != filter.Value {
				return false
			}
		case repositories.FieldStoreOrder:
			if order.StoreOrder != filter.Value {
				return false
			}
		case repositories.FieldStatus:
			if string(order.Status) != filter.Value {
				return false
			}
		case repositories.FieldUpdatedAt:
			if order.UpdatedAt.After(filter.Value.(time.Time)) {
				return false
			}
		}
	}
	return true
}

// shelfDatabase Opens only the Orders: these Tests Move Orders, never Place them.
type shelfDatabase struct{ orders *shelf }

func (d shelfDatabase) Collections() repositories.Collections {
	return repositories.Collections{Orders: d.orders}
}

func (shelfDatabase) Transact(context.Context, func(context.Context, repositories.Collections) error) error {
	return errors.New("the shelf keeps no transactions")
}

func (shelfDatabase) CloseStore() error { return nil }

func shelveOrders(orders ...model.Order) *repositories.OrderStore {
	kept := &shelf{kept: map[string]model.Order{}}
	for _, order := range orders {
		kept.kept[order.ID] = order
	}
	return repositories.NewOrderStore(shelfDatabase{orders: kept}, time.Hour)
}

func TestAConfirmThatArrivesAfterTheReleaseLoses(t *testing.T) {
	store := shelveOrders(model.Order{ID: "order_one", Status: model.OrderReleased})
	_, err := store.MoveOrderStatus(context.Background(), "order_one", model.OrderPending, model.OrderConfirmed)
	if !errors.Is(err, model.ErrOrderConflict) {
		t.Fatalf("late confirm err=%v", err)
	}
	kept, _ := store.GetOrder(context.Background(), "order_one")
	if kept.Status != model.OrderReleased {
		t.Fatalf("status=%s", kept.Status)
	}
}

func TestAMissingOrderAnswersItsOwnFailure(t *testing.T) {
	store := shelveOrders()
	if _, err := store.GetOrder(context.Background(), "order_missing"); !errors.Is(err, model.ErrOrderNotFound) {
		t.Fatalf("get err=%v", err)
	}
	if _, err := store.ReportOrder(context.Background(), "order_missing", "1001"); !errors.Is(err, model.ErrOrderNotFound) {
		t.Fatalf("report err=%v", err)
	}
	if _, err := store.FindOrderByStoreOrder(context.Background(), "konohastore.cl", "1001"); !errors.Is(err, model.ErrOrderNotFound) {
		t.Fatalf("find err=%v", err)
	}
}

func TestOnlyALinkedOrderTakesTheBuyersNumber(t *testing.T) {
	store := shelveOrders(
		model.Order{ID: "order_linked", Domain: "konohastore.cl", Status: model.OrderLinked},
		model.Order{ID: "order_pending", Domain: "konohastore.cl", Status: model.OrderPending},
	)
	reported, err := store.ReportOrder(context.Background(), "order_linked", "1001")
	if err != nil || reported.Status != model.OrderReported || reported.StoreOrder != "1001" {
		t.Fatalf("report=%#v err=%v", reported, err)
	}
	if _, err := store.ReportOrder(context.Background(), "order_pending", "1002"); !errors.Is(err, model.ErrOrderConflict) {
		t.Fatalf("pending report err=%v", err)
	}
	found, err := store.FindOrderByStoreOrder(context.Background(), "konohastore.cl", "1001")
	if err != nil || found.ID != "order_linked" {
		t.Fatalf("found=%#v err=%v", found, err)
	}
}

func TestTheSweepReleasesOnlyWhatIsStalePending(t *testing.T) {
	old := time.Now().UTC().Add(-48 * time.Hour)
	store := shelveOrders(
		model.Order{ID: "order_stale", Status: model.OrderPending, UpdatedAt: old},
		model.Order{ID: "order_fresh", Status: model.OrderPending, UpdatedAt: time.Now().UTC()},
		model.Order{ID: "order_done", Status: model.OrderConfirmed, UpdatedAt: old},
	)
	released, err := store.ReleaseExpiredOrders(context.Background(), 24*time.Hour, 10)
	if err != nil || released != 1 {
		t.Fatalf("released=%d err=%v", released, err)
	}
	for id, want := range map[string]model.OrderStatus{
		"order_stale": model.OrderReleased, "order_fresh": model.OrderPending, "order_done": model.OrderConfirmed,
	} {
		if got, _ := store.GetOrder(context.Background(), id); got.Status != want {
			t.Fatalf("%s status=%s want %s", id, got.Status, want)
		}
	}
}
