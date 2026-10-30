package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/repositories"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

func TestACollectionRefusesAnIdItAlreadyKeeps(t *testing.T) {
	orders := openTestStore(t).Collections().Orders
	ctx := context.Background()
	order := model.Order{ID: "order_one", Domain: "konohastore.cl", Status: model.OrderPending}
	if err := orders.Create(ctx, order.ID, order); err != nil {
		t.Fatal(err)
	}
	if err := orders.Create(ctx, order.ID, order); !errors.Is(err, repositories.ErrExists) {
		t.Fatalf("second create err=%v", err)
	}
	if _, err := orders.Get(ctx, "order_missing"); !errors.Is(err, repositories.ErrMissing) {
		t.Fatalf("get missing err=%v", err)
	}
}

func TestAnUpdateThatFailsLeavesTheRecordAlone(t *testing.T) {
	orders := openTestStore(t).Collections().Orders
	ctx := context.Background()
	order := model.Order{ID: "order_two", Domain: "konohastore.cl", Status: model.OrderPending}
	if err := orders.Create(ctx, order.ID, order); err != nil {
		t.Fatal(err)
	}
	refusal := errors.New("refused")
	_, err := orders.Update(ctx, order.ID, func(order model.Order) (model.Order, error) {
		order.Status = model.OrderConfirmed
		return order, refusal
	})
	if !errors.Is(err, refusal) {
		t.Fatalf("update err=%v", err)
	}
	kept, err := orders.Get(ctx, order.ID)
	if err != nil || kept.Status != model.OrderPending {
		t.Fatalf("kept=%#v err=%v", kept, err)
	}
	if _, err := orders.Update(ctx, "order_missing", func(order model.Order) (model.Order, error) { return order, nil }); !errors.Is(err, repositories.ErrMissing) {
		t.Fatalf("update missing err=%v", err)
	}
}

func TestAListFiltersOrdersAndCaps(t *testing.T) {
	orders := openTestStore(t).Collections().Orders
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour)
	for index, domain := range []string{"konohastore.cl", "konohastore.cl", "otherstore.cl"} {
		order := model.Order{
			ID: "order_" + string(rune('a'+index)), Domain: domain, Status: model.OrderPending,
			UpdatedAt: base.Add(time.Duration(index) * time.Minute),
		}
		if err := orders.Create(ctx, order.ID, order); err != nil {
			t.Fatal(err)
		}
	}
	listed, err := orders.List(ctx, repositories.Query{
		Filters: []repositories.Filter{{Field: repositories.FieldDomain, Operator: repositories.Equal, Value: "konohastore.cl"}},
		Limit:   1,
	})
	if err != nil || len(listed) != 1 || listed[0].Domain != "konohastore.cl" {
		t.Fatalf("filtered listed=%#v err=%v", listed, err)
	}
	listed, err = orders.List(ctx, repositories.Query{OrderBy: repositories.FieldUpdatedAt})
	if err != nil || len(listed) != 3 || listed[0].ID != "order_a" || listed[2].ID != "order_c" {
		t.Fatalf("ordered listed=%#v err=%v", listed, err)
	}
}

func TestAnOfferReadsBackUntilItsEntryExpires(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	offer := model.Offer{ID: "konohastore.cl:7", Source: "konohastore.cl"}
	if err := store.SaveOffers(ctx, "query-key", []model.Offer{offer}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if read, err := store.GetOffer(ctx, offer.ID); err != nil || read.ID != offer.ID {
		t.Fatalf("read=%#v err=%v", read, err)
	}
	if _, err := store.GetOffer(ctx, "konohastore.cl:8"); !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("unknown offer err=%v", err)
	}
	if err := store.SaveOffers(ctx, "stale-key", []model.Offer{{ID: "konohastore.cl:9"}}, -time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetOffer(ctx, "konohastore.cl:9"); !errors.Is(err, search.ErrNotFound) {
		t.Fatalf("expired offer err=%v", err)
	}
}

func TestADroppedEntryIsLoadedNoMore(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	if err := store.SaveOffers(ctx, "dropped-key", []model.Offer{{ID: "konohastore.cl:1"}}, time.Hour); err != nil {
		t.Fatal(err)
	}
	if err := store.DropOffers(ctx, "dropped-key"); err != nil {
		t.Fatal(err)
	}
	if _, found, err := store.LoadOffers(ctx, "dropped-key"); err != nil || found {
		t.Fatalf("found=%v err=%v", found, err)
	}
	if err := store.DropOffers(ctx, "never-kept"); err != nil {
		t.Fatalf("dropping a missing entry err=%v", err)
	}
}

func TestSourcesListByName(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	for _, domain := range []string{"zeta.cl", "alpha.cl", "mid.cl"} {
		if err := store.RecordSource(ctx, domain, 10*time.Millisecond, nil); err != nil {
			t.Fatal(err)
		}
	}
	listed, err := store.ListSourceHealth(ctx)
	if err != nil || len(listed) != 3 || listed[0].Source != "alpha.cl" || listed[2].Source != "zeta.cl" {
		t.Fatalf("listed=%#v err=%v", listed, err)
	}
}
