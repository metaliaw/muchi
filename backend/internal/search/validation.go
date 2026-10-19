package search

import (
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

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
