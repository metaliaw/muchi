package search

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

type Worker struct {
	Store           SearchItemRepository
	Service         Service
	Owner           string
	LeaseDuration   time.Duration
	PollInterval    time.Duration
	StockCheckLimit int
}

func (w Worker) RunWorker(ctx context.Context) error {
	for {
		err := w.processNext(ctx)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return err
		}
		if err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(w.PollInterval):
		}
	}
}

func (w Worker) processNext(ctx context.Context) error {
	item, err := w.Store.ClaimSearchItem(ctx, w.Owner, w.LeaseDuration)
	if err != nil {
		return err
	}
	work, cancel := context.WithCancel(ctx)
	defer cancel()
	go w.renewLease(work, item.ID)

	query := model.CardQuery{Name: item.NormalizedName, Match: item.Match, Kind: item.Kind}
	items, faults, sourceErr := w.Service.FindCardOffers(work, item.Game, query)
	if item.VerifyStock {
		items = w.verifyStocks(work, items)
	}
	item.Faults = faults
	item = applyItemResult(item, items, sourceErr)
	return w.Store.CompleteSearchItem(ctx, item, items)
}

// ProcessNext Handles one queued Card Task.
func (w Worker) ProcessNext(ctx context.Context) error {
	return w.processNext(ctx)
}

func applyItemResult(item model.Item, items []model.Offer, sourceErr error) model.Item {
	if len(items) > 0 {
		item.Source = items[0].Source
	}
	item.Status, item.ErrorCode, item.ErrorMessage = classifyResult(items, sourceErr)
	return item
}

func (w Worker) renewLease(ctx context.Context, itemID string) {
	interval := max(w.LeaseDuration/3, time.Second)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Store.RenewItemLease(ctx, itemID, w.Owner, w.LeaseDuration); err != nil {
				return
			}
		}
	}
}

func (w Worker) verifyStocks(ctx context.Context, items []model.Offer) []model.Offer {
	if w.Service.Stocks == nil {
		return items
	}
	targets := model.SelectStockOffers(items, w.StockCheckLimit)
	var group sync.WaitGroup
	for _, target := range targets {
		group.Add(1)
		go func(id string) {
			defer group.Done()
			w.checkOffer(ctx, items, id)
		}(target.ID)
	}
	group.Wait()
	return items
}

func (w Worker) checkOffer(ctx context.Context, items []model.Offer, id string) {
	for index := range items {
		if items[index].ID != id {
			continue
		}
		reading, err := w.Service.Stocks.CheckStock(ctx, items[index])
		if err == nil {
			items[index].StockStatus = reading.Status
			items[index].StockQuantity = reading.Quantity
		}
		return
	}
}

func classifyResult(items []model.Offer, err error) (model.ItemStatus, string, string) {
	if len(items) > 0 {
		return model.ItemFound, "", ""
	}
	if err != nil {
		return model.ItemSourceError, "source_unavailable", "Source unavailable"
	}
	return model.ItemNotFound, "", ""
}
