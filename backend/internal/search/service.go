package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

const maxConcurrentSources = 4

type Service struct {
	Searches          SearchStore
	Providers         map[Game]Provider
	Sources           []OfferSource
	SourcesByGame     map[Game][]OfferSource
	Stocks            StockChecker
	Cache             OfferCache
	CacheNamespace    string
	Tasks             TaskQueue
	CacheTTL          time.Duration
	EmptyCacheTTL     time.Duration
	MaxCards          int
	MaxQuantity       int
	SuspiciousPercent int
}

func (s Service) CreateSearch(ctx context.Context, key string, input CreateInput) (Job, error) {
	if err := ValidateCreate(input, s.MaxCards, s.MaxQuantity); err != nil {
		return Job{}, err
	}
	hash := HashPayload(input)
	job, err := s.Searches.CreateSearch(ctx, key, hash, input)
	if err != nil || s.Tasks == nil {
		return job, err
	}
	return job, s.Tasks.DispatchSearch(ctx, job)
}

func (s Service) GetSearch(ctx context.Context, id string) (Job, error) {
	return s.Searches.GetSearch(ctx, id)
}

func (s Service) ListResults(ctx context.Context, id string, page ResultPage) (Result, error) {
	return s.Searches.ListResults(ctx, id, page)
}

func (s Service) CancelSearch(ctx context.Context, id, key string) (Job, error) {
	hash := HashPayload(map[string]string{"search_id": id, "action": "cancel"})
	return s.Searches.CancelSearch(ctx, id, key, hash)
}

func (s Service) FindCardOffers(ctx context.Context, game Game, query offer.CardQuery) ([]offer.Offer, error) {
	query.Name = strings.TrimSpace(query.Name)
	if query.Name == "" || !validGame(game) {
		return nil, ErrInvalid
	}
	return s.collectOffers(ctx, game, query)
}

// collectOffers Keys the Cache by Match Mode too: a wide Answer must never
// Serve a narrow Question.
func (s Service) collectOffers(ctx context.Context, game Game, query offer.CardQuery) ([]offer.Offer, error) {
	key := s.CacheNamespace + string(game) + ":" + string(query.Match) + ":" + offer.NormalizeCard(query.Name)
	if items, found := s.loadOfferCache(ctx, key); found {
		return items, nil
	}
	sources := s.SourcesByGame[game]
	if len(sources) == 0 {
		sources = s.Sources
	}
	provider, hasProvider := s.Providers[game]
	if !hasProvider && len(sources) == 0 {
		return nil, ErrInvalid
	}
	items, err := queryGameSources(ctx, provider, hasProvider, sources, query)
	if err != nil && len(items) == 0 {
		return nil, err
	}
	items = offer.MarkSuspicious(offer.DeduplicateOffers(items), s.SuspiciousPercent)
	if err == nil {
		s.saveOfferCache(ctx, key, items)
	}
	return items, nil
}

func queryGameSources(ctx context.Context, provider Provider, hasProvider bool, sources []OfferSource, query offer.CardQuery) ([]offer.Offer, error) {
	results := make([][]offer.Offer, 0, len(sources)+1)
	errors := make([]error, 0, len(sources)+1)
	if hasProvider {
		items, err := provider.Search(ctx, query)
		results = append(results, items)
		errors = append(errors, err)
	}
	if len(sources) > 0 {
		items, err := querySources(ctx, sources, query)
		results = append(results, items)
		errors = append(errors, err)
	}
	return combineSources(results, errors)
}

func (s Service) loadOfferCache(ctx context.Context, key string) ([]offer.Offer, bool) {
	if s.Cache == nil {
		return nil, false
	}
	items, found, err := s.Cache.LoadOffers(ctx, key)
	return items, err == nil && found
}

func (s Service) saveOfferCache(ctx context.Context, key string, items []offer.Offer) {
	if s.Cache == nil {
		return
	}
	ttl := s.CacheTTL
	if len(items) == 0 {
		ttl = s.EmptyCacheTTL
	}
	_ = s.Cache.SaveOffers(ctx, key, items, ttl)
}

// ValidateCreate Bounds one Search Request by the configured Card and Quantity Limits.
func ValidateCreate(input CreateInput, maxCards, maxQuantity int) error {
	if !validGame(input.Game) || len(input.Cards) == 0 || len(input.Cards) > maxCards {
		return ErrInvalid
	}
	if _, err := offer.ReadMatchMode(string(input.Options.Match)); err != nil {
		return ErrInvalid
	}
	for _, card := range input.Cards {
		if strings.TrimSpace(card.Name) == "" || card.Quantity < 1 || card.Quantity > maxQuantity {
			return ErrInvalid
		}
	}
	return nil
}

func validGame(game Game) bool {
	return game == GameMagic || game == GamePokemon || game == GameYuGiOh || game == GameOnePiece
}

func HashPayload(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func querySources(ctx context.Context, sources []OfferSource, query offer.CardQuery) ([]offer.Offer, error) {
	results := make([][]offer.Offer, len(sources))
	errors := make([]error, len(sources))
	turns := make(chan struct{}, maxConcurrentSources)
	var group sync.WaitGroup
	for index, source := range sources {
		group.Add(1)
		go querySource(ctx, &group, turns, source, query, &results[index], &errors[index])
	}
	group.Wait()
	return combineSources(results, errors)
}

func querySource(ctx context.Context, group *sync.WaitGroup, turns chan struct{}, source OfferSource, query offer.CardQuery, items *[]offer.Offer, sourceErr *error) {
	defer group.Done()
	select {
	case turns <- struct{}{}:
		defer func() { <-turns }()
	case <-ctx.Done():
		*sourceErr = ctx.Err()
		return
	}
	*items, *sourceErr = source.FindOffers(ctx, query)
}

func combineSources(results [][]offer.Offer, errors []error) ([]offer.Offer, error) {
	var result []offer.Offer
	var lastErr error
	for index, items := range results {
		if errors[index] != nil {
			lastErr = errors[index]
		}
		result = append(result, items...)
	}
	return result, lastErr
}
