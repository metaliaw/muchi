package search

import (
	"context"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// OfferService Finds Offers for one Card across the Providers and Sources of
// its Game, Caches the Answer and Shapes it before it Leaves.
type OfferService struct {
	Providers         map[model.Game]Provider
	Sources           []OfferSource
	SourcesByGame     map[model.Game][]OfferSource
	Cache             OfferCache
	Offers            OfferReader
	CacheNamespace    string
	CacheTTL          time.Duration
	EmptyCacheTTL     time.Duration
	SuspiciousPercent int
	StoreLocations    map[string][]string
	// SetsByGame Lends a Game its Sets, so a Sealed Answer can Tell whether a
	// Box Belongs to the Game that was Asked about.
	SetsByGame map[model.Game]SetLibrary
	// PrintsByGame Lends a Card its Printings. Only a Game with a Catalog of
	// Images Appears here; the rest Keep whatever Image their Source Sent.
	PrintsByGame map[model.Game]PrintLibrary
	// SinglesOnly Names the Sources that Sell no Unopened Product, as their
	// Configuration Declares it. A Sealed Question Skips them: asking a Singles
	// Index for a Booster Box Spends a Request, Waits out its Timeout and
	// Answers nothing — and that Wait Comes back Marked as an incomplete
	// Answer, which Reads to a Caller like the Box might Exist elsewhere.
	SinglesOnly map[string]bool
	// SealedOnly Names the Sources that Sell no Loose Card, the Mirror of
	// SinglesOnly. A Question for a Card Skips them for the same Reason, and
	// with a Worse Failure if it does not: a Shop that Sells only Boxes
	// Answers "Mazo de 50 Cartas" to a Card Name, and that Box Looks like an
	// Offer for the Card until someone Reads the Title.
	SealedOnly map[string]bool
}

func (s OfferService) FindCardOffers(ctx context.Context, game model.Game, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
	query.Name = strings.TrimSpace(query.Name)
	if query.Name == "" || !validGame(game) {
		return nil, nil, ErrInvalid
	}
	return s.collectOffers(ctx, game, query)
}

// GetOffer Reads the Latest Cached Snapshot for one Stable Offer Id.
func (s OfferService) GetOffer(ctx context.Context, id string) (model.Offer, error) {
	if strings.TrimSpace(id) == "" || s.Offers == nil {
		return model.Offer{}, ErrInvalid
	}
	return s.Offers.GetOffer(ctx, id)
}

// collectOffers Keys the Cache by Match Mode and Product Kind too: a wide
// Answer must never Serve a narrow Question, and a Booster Box must never
// Answer the Question about the Card Printed inside it.
func (s OfferService) collectOffers(ctx context.Context, game model.Game, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
	query = settleQuery(query)
	key := s.CacheNamespace + string(game) + ":" + string(query.Kind) + ":" + string(query.Match) + ":" + model.NormalizeCard(query.Name)
	if items, found := s.loadOfferCache(ctx, key); found {
		return items, nil, nil
	}
	sources := s.SourcesByGame[game]
	if len(sources) == 0 {
		sources = s.Sources
	}
	provider, hasProvider := s.Providers[game]
	if !hasProvider && len(sources) == 0 {
		return nil, nil, ErrInvalid
	}
	if query.Sealed() {
		sources = s.keepSources(sources, s.SinglesOnly)
		hasProvider = hasProvider && !s.SinglesOnly[provider.SourceName()]
	} else {
		sources = s.keepSources(sources, s.SealedOnly)
		hasProvider = hasProvider && !s.SealedOnly[provider.SourceName()]
	}
	items, faults, err := queryGameSources(ctx, provider, hasProvider, sources, query)
	if err != nil && len(items) == 0 {
		return nil, faults, err
	}
	items = model.NameCards(nameKinds(model.DeduplicateOffers(items), query.Kind))
	items = s.keepGameSealed(ctx, game, query, items)
	items = s.applyStoreLocations(items)
	items = s.applyPrintImages(ctx, game, query, items)
	items = model.MarkSuspicious(items, s.SuspiciousPercent, query)
	if err == nil {
		s.saveOfferCache(ctx, key, items)
	}
	return items, faults, nil
}

func (s OfferService) loadOfferCache(ctx context.Context, key string) ([]model.Offer, bool) {
	if s.Cache == nil {
		return nil, false
	}
	items, found, err := s.Cache.LoadOffers(ctx, key)
	return items, err == nil && found
}

func (s OfferService) saveOfferCache(ctx context.Context, key string, items []model.Offer) {
	if s.Cache == nil {
		return
	}
	ttl := s.CacheTTL
	if len(items) == 0 {
		ttl = s.EmptyCacheTTL
	}
	_ = s.Cache.SaveOffers(ctx, key, items, ttl)
}
