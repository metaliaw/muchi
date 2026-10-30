package search

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
)

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
func (s OfferService) keepSources(sources []OfferSource, excluded map[string]bool) []OfferSource {
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
func (s OfferService) keepGameSealed(ctx context.Context, game model.Game, query model.CardQuery, items []model.Offer) []model.Offer {
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
func (s OfferService) applyStoreLocations(items []model.Offer) []model.Offer {
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
func (s OfferService) applyPrintImages(ctx context.Context, game model.Game, query model.CardQuery, items []model.Offer) []model.Offer {
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
