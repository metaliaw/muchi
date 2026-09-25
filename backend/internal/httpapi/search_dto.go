package httpapi

import (
	"net/http"

	"github.com/cangrejometralleta/muchi-api/internal/model"
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

func decodeSearch(r *http.Request) (model.CreateInput, error) {
	var body createSearchBody
	if err := decodeJSON(r, &body); err != nil {
		return model.CreateInput{}, err
	}
	if _, err := model.ReadMatchMode(body.Options.Match); err != nil {
		return model.CreateInput{}, search.ErrInvalid
	}
	if _, err := model.ReadProductKind(body.Options.Kind); err != nil {
		return model.CreateInput{}, search.ErrInvalid
	}
	input := model.CreateInput{
		Game: model.Game(body.Game),
		Options: model.Options{
			VerifyStock: body.Options.VerifyStock,
			Match:       model.MatchMode(body.Options.Match),
			Kind:        model.ProductKind(body.Options.Kind),
		},
		Cards: make([]model.CardInput, 0, len(body.Cards)),
	}
	if input.Game == "" {
		input.Game = model.GameMagic
	}
	for _, card := range body.Cards {
		input.Cards = append(input.Cards, model.CardInput{Name: card.Name, Quantity: card.Quantity})
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

func decodeOfferQuery(r *http.Request) (model.Game, model.CardQuery, error) {
	values := r.URL.Query()
	query := offerQuery{
		Game: values.Get("game"), Name: values.Get("name"),
		Match: values.Get("match"), Kind: values.Get("kind"),
	}
	game := model.Game(query.Game)
	if game == "" {
		game = model.GameMagic
	}
	match, err := model.ReadMatchMode(query.Match)
	if err != nil {
		return "", model.CardQuery{}, search.ErrInvalid
	}
	kind, err := model.ReadProductKind(query.Kind)
	if err != nil {
		return "", model.CardQuery{}, search.ErrInvalid
	}
	return game, model.CardQuery{Name: query.Name, Match: match, Kind: kind}, nil
}
