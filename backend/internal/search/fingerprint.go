package search

import "github.com/cangrejometralleta/muchi-api/internal/model"

// searchFingerprint Preserves the idempotency representation across model changes.
type searchFingerprint struct {
	Game    model.Game         `json:"game"`
	Cards   []cardFingerprint  `json:"cards"`
	Options optionsFingerprint `json:"options"`
}
type cardFingerprint struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}
type optionsFingerprint struct {
	VerifyStock bool              `json:"verify_stock"`
	Match       model.MatchMode   `json:"match,omitempty"`
	Kind        model.ProductKind `json:"kind,omitempty"`
}

func renderFingerprint(input model.CreateInput) searchFingerprint {
	var cards []cardFingerprint
	if input.Cards != nil {
		cards = make([]cardFingerprint, len(input.Cards))
		for i, card := range input.Cards {
			cards[i] = cardFingerprint{Name: card.Name, Quantity: card.Quantity}
		}
	}
	return searchFingerprint{Game: input.Game, Cards: cards, Options: optionsFingerprint{
		VerifyStock: input.Options.VerifyStock, Match: input.Options.Match, Kind: input.Options.Kind,
	}}
}
