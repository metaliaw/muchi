package httpapi

import (
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// createSearchBody is the untrusted HTTP shape of a search request.
type createSearchBody struct {
	Game    string            `json:"game"`
	Cards   []searchCardBody  `json:"cards"`
	Options searchOptionsBody `json:"options"`
}

type searchCardBody struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type searchOptionsBody struct {
	VerifyStock bool   `json:"verify_stock"`
	Match       string `json:"match"`
	Kind        string `json:"kind"`
}

func decodeSearch(r *http.Request) (search.CreateInput, error) {
	var body createSearchBody
	if err := decodeJSON(r, &body); err != nil {
		return search.CreateInput{}, err
	}
	if _, err := offer.ReadMatchMode(body.Options.Match); err != nil {
		return search.CreateInput{}, search.ErrInvalid
	}
	if _, err := offer.ReadProductKind(body.Options.Kind); err != nil {
		return search.CreateInput{}, search.ErrInvalid
	}
	input := search.CreateInput{
		Game: search.Game(body.Game),
		Options: search.Options{
			VerifyStock: body.Options.VerifyStock,
			Match:       offer.MatchMode(body.Options.Match),
			Kind:        offer.ProductKind(body.Options.Kind),
		},
		Cards: make([]search.CardInput, 0, len(body.Cards)),
	}
	if input.Game == "" {
		input.Game = search.GameMagic
	}
	for _, card := range body.Cards {
		input.Cards = append(input.Cards, search.CardInput{Name: card.Name, Quantity: card.Quantity})
	}
	return input, nil
}

// offerQuery is the HTTP shape before match and kind become business values.
type offerQuery struct {
	Game  string
	Name  string
	Match string
	Kind  string
}

func decodeOfferQuery(r *http.Request) (search.Game, offer.CardQuery, error) {
	values := r.URL.Query()
	query := offerQuery{
		Game: values.Get("game"), Name: values.Get("name"),
		Match: values.Get("match"), Kind: values.Get("kind"),
	}
	game := search.Game(query.Game)
	if game == "" {
		game = search.GameMagic
	}
	match, err := offer.ReadMatchMode(query.Match)
	if err != nil {
		return "", offer.CardQuery{}, search.ErrInvalid
	}
	kind, err := offer.ReadProductKind(query.Kind)
	if err != nil {
		return "", offer.CardQuery{}, search.ErrInvalid
	}
	return game, offer.CardQuery{Name: query.Name, Match: match, Kind: kind}, nil
}
