package search

import (
	"context"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TestACardQuestionSkipsASealedOnlySource Covers the Mirror Mark. A Shop that
// Sells only Boxes Answers a Card Name with "Mazo de 50 Cartas", and that Box
// Reads like an Offer for the Card until someone Reads the Title — a worse
// Failure than the empty Answer the Singles Mark Prevents.
func TestACardQuestionSkipsASealedOnlySource(t *testing.T) {
	boxes := &countingSource{name: "casamyl.cl"}
	cards := &countingSource{name: "cards.example.cl"}
	service := Service{
		SourcesByGame: map[model.Game][]OfferSource{model.GameMitos: {boxes, cards}},
		SealedOnly:    map[string]bool{"casamyl.cl": true},
	}

	if _, _, err := service.FindCardOffers(context.Background(), model.GameMitos,
		model.CardQuery{Name: "Dragón de Magma", Kind: model.KindSingle}); err != nil {
		t.Fatal(err)
	}
	if boxes.asked != 0 {
		t.Errorf("a sealed-only source was asked for a card %d times", boxes.asked)
	}
	if cards.asked != 1 {
		t.Errorf("the singles source was asked %d times", cards.asked)
	}

	// La Caja sí es su Pregunta: la Marca Saca la Tienda de una Lista, no de
	// la Búsqueda.
	if _, _, err := service.FindCardOffers(context.Background(), model.GameMitos,
		model.CardQuery{Name: "Display Espada Sagrada", Kind: model.KindSealed}); err != nil {
		t.Fatal(err)
	}
	if boxes.asked != 1 {
		t.Errorf("the sealed source was asked %d times for a box", boxes.asked)
	}
}
