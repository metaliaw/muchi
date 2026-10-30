package model

import "testing"

// Mitos y Leyendas Names its Cards in Spanish; its Index Files them without
// Tildes. Sin Plegar el Acento, la Carta que se Busca no es la que se Ve.
func TestATildeIsNotAnotherCard(t *testing.T) {
	if !MatchesCard("dragon de magma", "Dragón de Magma") {
		t.Fatal("the accented name missed its own card")
	}
	if !ContainsCard("Dragón de Magma [Imperio]", "dragon de magma") {
		t.Fatal("the unaccented name missed the accented title")
	}
	if NormalizeCard("Pokémon") != "pokemon" {
		t.Fatalf("normalized=%q", NormalizeCard("Pokémon"))
	}
}
