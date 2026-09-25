package search

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type oneItemStore struct {
	item      model.Item
	claimed   atomic.Bool
	completed model.Item
	offers    []model.Offer
}

func (s *oneItemStore) ClaimSearchItem(context.Context, string, time.Duration) (model.Item, error) {
	if s.claimed.Swap(true) {
		return model.Item{}, ErrNotFound
	}
	return s.item, nil
}

func (s *oneItemStore) CompleteSearchItem(_ context.Context, item model.Item, items []model.Offer) error {
	s.completed, s.offers = item, items
	return nil
}

func (s *oneItemStore) RenewItemLease(context.Context, string, string, time.Duration) error {
	return nil
}
func (s *oneItemStore) CreateSearch(context.Context, string, string, model.CreateInput) (model.Job, error) {
	return model.Job{}, nil
}
func (s *oneItemStore) GetSearch(context.Context, string) (model.Job, error) { return model.Job{}, nil }
func (s *oneItemStore) CancelSearch(context.Context, string, string, string) (model.Job, error) {
	return model.Job{}, nil
}
func (s *oneItemStore) ListResults(context.Context, string, model.ResultPage) (model.Result, error) {
	return model.Result{}, nil
}

type countingChecker struct{ calls atomic.Int32 }

func (c *countingChecker) CheckStock(context.Context, model.Offer) (model.StockReading, error) {
	c.calls.Add(1)
	return model.CountStock("unavailable", 0), nil
}

func workerFor(item model.Item, checker StockChecker) (Worker, *oneItemStore) {
	store := &oneItemStore{item: item}
	return Worker{
		Store: store, Owner: "probe", LeaseDuration: time.Minute, StockCheckLimit: 5,
		Service: Service{
			Stocks: checker,
			SourcesByGame: map[model.Game][]OfferSource{model.GameMagic: {stubSource{name: "store.cl", items: []model.Offer{
				{ID: "one", CardName: "Sol Ring", Store: "store", URL: "https://store.test/one",
					PriceAmount: "2800", PriceCurrency: "CLP", StockStatus: "available"},
			}}}},
		},
	}, store
}

// TestTheWorkerObeysVerifyStock Closes the Chain the Storage Test Opens: the
// Option Survives the Round Trip, and the Worker Acts on what it Read.
func TestTheWorkerObeysVerifyStock(t *testing.T) {
	checker := &countingChecker{}
	worker, store := workerFor(model.Item{
		ID: "item-1", Game: model.GameMagic, NormalizedName: "sol ring",
		Status: model.ItemPending, VerifyStock: true,
	}, checker)
	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if checker.calls.Load() != 1 {
		t.Fatalf("checked stock %d times, want 1", checker.calls.Load())
	}
	if store.offers[0].StockStatus != "unavailable" {
		t.Fatalf("the check did not reach the offer: %q", store.offers[0].StockStatus)
	}
}

// TestTheWorkerSkipsTheCheckWhenNobodyAsked Keeps the Calls off by Default.
func TestTheWorkerSkipsTheCheckWhenNobodyAsked(t *testing.T) {
	checker := &countingChecker{}
	worker, _ := workerFor(model.Item{
		ID: "item-1", Game: model.GameMagic, NormalizedName: "sol ring", Status: model.ItemPending,
	}, checker)
	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatal(err)
	}
	if checker.calls.Load() != 0 {
		t.Fatalf("checked stock %d times without being asked", checker.calls.Load())
	}
}
