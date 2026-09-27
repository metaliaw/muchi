package search

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

const maxConcurrentSources = 4

type Service struct {
	Repository        SearchRepository
	Providers         map[model.Game]Provider
	Sources           []OfferSource
	SourcesByGame     map[model.Game][]OfferSource
	Stocks            StockChecker
	Checkouts         CheckoutLinker
	Quotes            CartQuoter
	Orderer           OrderPlacer
	Orders            OrderRepository
	Cache             OfferCache
	CacheNamespace    string
	Tasks             TaskQueue
	CacheTTL          time.Duration
	EmptyCacheTTL     time.Duration
	MaxCards          int
	MaxQuantity       int
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

func (s Service) CreateSearch(ctx context.Context, key string, input model.CreateInput) (model.Job, error) {
	if err := ValidateCreate(input, s.MaxCards, s.MaxQuantity); err != nil {
		return model.Job{}, err
	}
	hash := HashPayload(input)
	job, err := s.Repository.CreateSearch(ctx, key, hash, input)
	if err != nil || s.Tasks == nil {
		return job, err
	}
	return job, s.Tasks.DispatchSearch(ctx, job)
}

func (s Service) GetSearch(ctx context.Context, id string) (model.Job, error) {
	return s.Repository.GetSearch(ctx, id)
}

func (s Service) ListResults(ctx context.Context, id string, page model.ResultPage) (model.Result, error) {
	return s.Repository.ListResults(ctx, id, page)
}

func (s Service) CancelSearch(ctx context.Context, id, key string) (model.Job, error) {
	hash := HashPayload(map[string]string{"search_id": id, "action": "cancel"})
	return s.Repository.CancelSearch(ctx, id, key, hash)
}

func (s Service) FindCardOffers(ctx context.Context, game model.Game, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
	query.Name = strings.TrimSpace(query.Name)
	if query.Name == "" || !validGame(game) {
		return nil, nil, ErrInvalid
	}
	return s.collectOffers(ctx, game, query)
}

// collectOffers Keys the Cache by Match Mode and Product Kind too: a wide
// Answer must never Serve a narrow Question, and a Booster Box must never
// Answer the Question about the Card Printed inside it.
func (s Service) collectOffers(ctx context.Context, game model.Game, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
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

// settleQuery Writes the Defaults down before anything Reads them.
//
// An Item Stored before a Field Existed Comes back Empty, and an Empty Kind
// Means the same as `single` — but not to a Cache Key, which Compares Strings
// and would Keep two Namespaces for one Question. The Zero Value is the right
// Answer; it just has to be Spelled.
func settleQuery(query model.CardQuery) model.CardQuery {
	if query.Kind == "" {
		query.Kind = model.KindSingle
	}
	if query.Match == "" {
		query.Match = model.MatchExact
	}
	return query
}

// nameKinds Tells every Offer what the Question was for. A Store Title alone
// cannot Say whether it Sells the Card or the Box: the Question can.
func nameKinds(items []model.Offer, kind model.ProductKind) []model.Offer {
	if kind == "" {
		kind = model.KindSingle
	}
	for position := range items {
		items[position].Kind = kind
	}
	return items
}

// keepSources Drops the Sources a Mark Excludes from this Kind of Question.
//
// It Fails open, like the Set Filter: a Source nobody Marked Stays Asked. The
// Mark is a Fact about a Shop, and an unmarked Shop is one nobody Looked at
// yet — not one that Sells nothing.
func (s Service) keepSources(sources []OfferSource, excluded map[string]bool) []OfferSource {
	if len(excluded) == 0 {
		return sources
	}
	kept := make([]OfferSource, 0, len(sources))
	for _, source := range sources {
		if !excluded[source.SourceName()] {
			kept = append(kept, source)
		}
	}
	return kept
}

// keepGameSealed Drops a Box that Belongs to another Game.
//
// `Booster Box` Names no Game, so a Store Selling several Answers with all of
// them: a Yu-Gi-Oh Question Came back with Cardfight!! Vanguard. A Sealed Title
// Carries its Set, and the Set Belongs to one Game — that is the Filter the
// Card Name Gives for free in a Singles Search and nothing Gives here.
//
// It Fails open. A Game without a Set List, or a List that would not Load,
// Answers what the Sources Sent: fewer Offers Hurt a Caller more than a
// Stranger among them.
func (s Service) keepGameSealed(ctx context.Context, game model.Game, query model.CardQuery, items []model.Offer) []model.Offer {
	library, found := s.SetsByGame[game]
	if !query.Sealed() || !found || library == nil || len(items) == 0 {
		return items
	}
	sets, err := library.GameSets(ctx)
	if err != nil || len(sets) == 0 {
		return items
	}
	kept := make([]model.Offer, 0, len(items))
	for _, item := range items {
		if model.NamesSet(item.CardName, sets) || model.NamesSet(item.Edition, sets) {
			kept = append(kept, item)
		}
	}
	return kept
}

// applyStoreLocations Adds only Locations verified in Store Configuration.
func (s Service) applyStoreLocations(items []model.Offer) []model.Offer {
	for position := range items {
		locations := s.StoreLocations[items[position].Source]
		if len(locations) == 0 {
			locations = s.StoreLocations[items[position].Store]
		}
		items[position].Locations = locations
	}
	return items
}

// applyPrintImages Gives an Offer the Picture of the Printing its Title Names.
// A Store Publishes a Title and a Price, not an Image; the Printing List has
// the Image and the Title Says which Printing. Only an Offer Arriving without
// one is Filled: a Source that Sent its own Picture Knows better.
//
// A Sealed Offer Never Gets one from here. The Library Lists Printed Cards, and
// `Bloomburrow` Names both a Set and the Cards in it: filling a Booster Box with
// the Picture of a Card Printed inside it Looks like an Answer and is a Lie.
// A Box Wears the Photo its Store Published, or none.
func (s Service) applyPrintImages(ctx context.Context, game model.Game, query model.CardQuery, items []model.Offer) []model.Offer {
	library, found := s.PrintsByGame[game]
	if query.Sealed() || !found || library == nil || !anyImageMissing(items) {
		return items
	}
	prints, err := library.CardPrints(ctx, query.Name)
	if err != nil {
		return items
	}
	index := cardmetadata.IndexPrints(prints)
	if index.Empty() {
		return items
	}
	for position := range items {
		if items[position].Image == "" {
			items[position].Image = index.ImageFor(readOfferTitle(items[position]), query.Name)
		}
	}
	return items
}

// readOfferTitle Prefers the Title the Store Published, because the Edition
// Lives there and not always in the Card Name.
func readOfferTitle(item model.Offer) string {
	if title := item.Metadata["title"]; title != "" {
		return title
	}
	return item.CardName
}

func anyImageMissing(items []model.Offer) bool {
	for _, item := range items {
		if item.Image == "" {
			return true
		}
	}
	return false
}

func queryGameSources(ctx context.Context, provider Provider, hasProvider bool, sources []OfferSource, query model.CardQuery) ([]model.Offer, []model.SourceFault, error) {
	var items []model.Offer
	var faults []model.SourceFault
	var lastErr error
	if hasProvider {
		found, err := provider.Search(ctx, query)
		items = append(items, found...)
		if err != nil {
			faults = append(faults, model.SourceFault{provider.SourceName(), err.Error()})
			lastErr = err
		}
	}
	if len(sources) > 0 {
		found, storeFaults, direct, err := querySources(ctx, sources, query)
		items = preferDirectOffers(items, direct)
		items = append(items, found...)
		faults = append(faults, storeFaults...)
		if err != nil {
			lastErr = err
		}
	}
	return items, faults, lastErr
}

// preferDirectOffers Drops Aggregated Offers when their Store Answered Directly.
func preferDirectOffers(items []model.Offer, direct map[string]bool) []model.Offer {
	kept := items[:0]
	for _, item := range items {
		if direct[readOfferHost(item)] {
			continue
		}
		kept = append(kept, item)
	}
	return kept
}

func readOfferHost(item model.Offer) string {
	link, err := url.Parse(item.URL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(link.Hostname()), "www.")
}

func (s Service) loadOfferCache(ctx context.Context, key string) ([]model.Offer, bool) {
	if s.Cache == nil {
		return nil, false
	}
	items, found, err := s.Cache.LoadOffers(ctx, key)
	return items, err == nil && found
}

func (s Service) saveOfferCache(ctx context.Context, key string, items []model.Offer) {
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
func ValidateCreate(input model.CreateInput, maxCards, maxQuantity int) error {
	if !validGame(input.Game) || len(input.Cards) == 0 || len(input.Cards) > maxCards {
		return ErrInvalid
	}
	if _, err := model.ReadMatchMode(string(input.Options.Match)); err != nil {
		return ErrInvalid
	}
	if _, err := model.ReadProductKind(string(input.Options.Kind)); err != nil {
		return ErrInvalid
	}
	for _, card := range input.Cards {
		if strings.TrimSpace(card.Name) == "" || card.Quantity < 1 || card.Quantity > maxQuantity {
			return ErrInvalid
		}
	}
	return nil
}

func validGame(game model.Game) bool {
	return game == model.GameMagic || game == model.GamePokemon || game == model.GameYuGiOh ||
		game == model.GameOnePiece || game == model.GameDigimon || game == model.GameRiftbound ||
		game == model.GameMitos
}

func HashPayload(value any) string {
	if input, ok := value.(model.CreateInput); ok {
		value = renderFingerprint(input)
	}
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func querySources(ctx context.Context, sources []OfferSource, query model.CardQuery) ([]model.Offer, []model.SourceFault, map[string]bool, error) {
	results := make([][]model.Offer, len(sources))
	errors := make([]error, len(sources))
	turns := make(chan struct{}, maxConcurrentSources)
	var group sync.WaitGroup
	for index, source := range sources {
		group.Add(1)
		go querySource(ctx, &group, turns, source, query, &results[index], &errors[index])
	}
	group.Wait()
	items, faults, err := combineSources(sources, results, errors)
	return items, faults, readDirectHosts(sources, errors), err
}

func readDirectHosts(sources []OfferSource, errors []error) map[string]bool {
	result := make(map[string]bool)
	for index, source := range sources {
		name := strings.ToLower(source.SourceName())
		if errors[index] == nil && strings.Contains(name, ".") && !strings.ContainsAny(name, ":/") {
			result[strings.TrimPrefix(name, "www.")] = true
		}
	}
	return result
}

func querySource(ctx context.Context, group *sync.WaitGroup, turns chan struct{}, source OfferSource, query model.CardQuery, items *[]model.Offer, sourceErr *error) {
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

// combineSources Keeps every Failure by Name. A Source that Fell while others
// Answered used to Vanish, and a partial Answer Looked exactly like a full one.
func combineSources(sources []OfferSource, results [][]model.Offer, errors []error) ([]model.Offer, []model.SourceFault, error) {
	var result []model.Offer
	var faults []model.SourceFault
	var lastErr error
	for index, items := range results {
		if errors[index] != nil {
			faults = append(faults, model.SourceFault{sources[index].SourceName(), errors[index].Error()})
			lastErr = errors[index]
		}
		result = append(result, items...)
	}
	return result, faults, lastErr
}
