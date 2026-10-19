package search

import (
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"testing"
)

func TestExistingRequestKeepsIdempotencyHash(t *testing.T) {
	input := model.CreateInput{Game: model.GameMagic, Cards: []model.CardInput{{Name: "Sol Ring", Quantity: 2}}, Options: model.Options{VerifyStock: true, Match: model.MatchIncludes, Kind: model.KindSealed}}
	const want = "947bd908e97406e60342cb305d00cd81851f09aee271c07c8993412941ce5231"
	if got := HashPayload(input); got != want {
		t.Fatalf("hash = %s, want %s", got, want)
	}
}
