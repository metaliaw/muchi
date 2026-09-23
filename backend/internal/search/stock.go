package search

import (
	"context"
	"sync"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// maxCheckedOffers Caps one Question. The Caller Asks for the cheapest Offer of
// each Card, not for a whole Page: a longer List would Visit Stores that no
// Recommendation Depends on.
const maxCheckedOffers = 50

// OfferStock Names the Offer the Reading Belongs to.
type OfferStock struct {
	ID string `json:"id"`
	offer.StockReading
}

// CheckOfferStock Asks the Stores again about Offers this Search already Found.
//
// The Price Survives the Search; the Stock does not. A Caller Crowning the
// cheapest Offer Needs to know whether that one is still There, and only the
// Store can Say. The Offer is Named by the Id the Search gave it, so nobody
// Sends a URL of their own choosing and Makes this a Proxy.
func (s Service) CheckOfferStock(ctx context.Context, id string, ids []string) ([]OfferStock, error) {
	if id == "" || len(ids) == 0 || len(ids) > maxCheckedOffers {
		return nil, ErrInvalid
	}
	if s.Stocks == nil {
		return nil, ErrInvalid
	}
	found, err := s.collectSearchOffers(ctx, id)
	if err != nil {
		return nil, err
	}
	readings := make([]OfferStock, len(ids))
	var group sync.WaitGroup
	gate := make(chan struct{}, maxConcurrentSources)
	for position, offerID := range ids {
		item, known := found[offerID]
		if !known {
			return nil, ErrNotFound
		}
		group.Add(1)
		go func(position int, item offer.Offer) {
			defer group.Done()
			gate <- struct{}{}
			defer func() { <-gate }()
			readings[position] = s.readOfferStock(ctx, item)
		}(position, item)
	}
	group.Wait()
	return readings, nil
}

// readOfferStock Keeps a Store Failure inside the Answer. One Store that will
// not Talk is not a failed Question: the Offer simply Stays unknown, and the
// Caller Moves to the next cheapest.
func (s Service) readOfferStock(ctx context.Context, item offer.Offer) OfferStock {
	reading, err := s.Stocks.CheckStock(ctx, item)
	if err != nil {
		return OfferStock{ID: item.ID, StockReading: offer.ReadStock("unknown")}
	}
	return OfferStock{ID: item.ID, StockReading: reading}
}

// collectSearchOffers Reads every Page of a Search and Indexes its Offers by Id.
func (s Service) collectSearchOffers(ctx context.Context, id string) (map[string]offer.Offer, error) {
	found := map[string]offer.Offer{}
	for cursor := 0; ; {
		page, err := s.Repository.ListResults(ctx, id, ResultPage{After: cursor, Limit: maxResultPage})
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			for _, candidate := range item.Offers {
				found[candidate.ID] = candidate
			}
		}
		if !page.HasMore || page.Cursor <= cursor {
			return found, nil
		}
		cursor = page.Cursor
	}
}

// maxResultPage Reads the Results the same Way the public Route Caps them.
const maxResultPage = 100
