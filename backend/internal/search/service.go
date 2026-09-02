package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

type Service struct {
	Searches SearchStore
	Sources  []OfferSource
	Stocks   StockChecker
	Cache    OfferCache
}

func (s Service) CreateSearch(ctx context.Context, key string, input CreateInput) (Job, error) {
	if err := ValidateCreate(input); err != nil {
		return Job{}, err
	}
	hash := HashPayload(input)
	return s.Searches.CreateSearch(ctx, key, hash, input)
}

func (s Service) GetSearch(ctx context.Context, id string) (Job, error) {
	return s.Searches.GetSearch(ctx, id)
}

func (s Service) ListResults(ctx context.Context, id string) (Result, error) {
	return s.Searches.ListResults(ctx, id)
}

func (s Service) CancelSearch(ctx context.Context, id, key string) (Job, error) {
	hash := HashPayload(map[string]string{"search_id": id, "action": "cancel"})
	return s.Searches.CancelSearch(ctx, id, key, hash)
}

func (s Service) FindCardOffers(ctx context.Context, name string) ([]offer.Offer, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalid
	}
	return s.collectOffers(ctx, name)
}

func (s Service) collectOffers(ctx context.Context, name string) ([]offer.Offer, error) {
	key := offer.NormalizeCard(name)
	if s.Cache != nil {
		if items, found, err := s.Cache.LoadOffers(ctx, key); err == nil && found {
			return items, nil
		}
	}
	items, err := querySources(ctx, s.Sources, name)
	if err != nil && len(items) == 0 {
		return nil, err
	}
	items = offer.MarkSuspicious(offer.DeduplicateOffers(items))
	if s.Cache != nil {
		ttl := 15 * time.Minute
		if len(items) == 0 {
			ttl = 2 * time.Minute
		}
		_ = s.Cache.SaveOffers(ctx, key, items, ttl)
	}
	return items, nil
}

func ValidateCreate(input CreateInput) error {
	if len(input.Cards) == 0 || len(input.Cards) > 500 {
		return ErrInvalid
	}
	for _, card := range input.Cards {
		if strings.TrimSpace(card.Name) == "" || card.Quantity < 1 || card.Quantity > 99 {
			return ErrInvalid
		}
	}
	return nil
}

func HashPayload(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func querySources(ctx context.Context, sources []OfferSource, name string) ([]offer.Offer, error) {
	var result []offer.Offer
	var lastErr error
	for _, source := range sources {
		items, err := source.FindOffers(ctx, name)
		if err != nil {
			lastErr = err
			continue
		}
		result = append(result, items...)
	}
	return result, lastErr
}
