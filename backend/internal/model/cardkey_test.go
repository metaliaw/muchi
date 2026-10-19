package model

import "testing"

// TestReadCardKeyNamesOneCard Covers the real Title Shapes the Sources Send.
func TestReadCardKeyNamesOneCard(t *testing.T) {
	for _, pair := range [][2]string{
		{"Winged Kuriboh", "winged kuriboh"},
		{"LDS3-EN100 “Winged Kuriboh” Common Effect Monster", "winged kuriboh"},
		{"LDS3-SP100 “Winged Kuriboh” Common Effect Monster (Español)", "winged kuriboh"},
		{"Astral Kuriboh (PUR)", "astral kuriboh"},
		{"RA04-EN040 “Astral Kuriboh” Secret Rare", "astral kuriboh"},
		{"Kuriboh - Multiply!", "kuriboh - multiply!"},
		{"MAMO-EN002 “Kuriboh – Multiply!” Ultra Rare Effect Monster", "kuriboh - multiply!"},
		{"Kuriboh (C)", "kuriboh"},
		{"Kuriboh", "kuriboh"},
		{"Token: Kuriboh (Orange)", "token: kuriboh"},
		{"Mewtwo - SM214", "mewtwo"},
		{"Sol Ring - 212 - uncommon", "sol ring"},
		{"Sol Ring (2683) [Secret Lair Drop Series]", "sol ring"},
	} {
		if got := ReadCardKey(pair[0]); got != pair[1] {
			t.Errorf("ReadCardKey(%q) = %q, want %q", pair[0], got, pair[1])
		}
	}
}

// TestOneCardFromThreeSources Covers what Production Showed: one Card Split in
// three because each Source Writes its Title its own Way.
func TestOneCardFromThreeSources(t *testing.T) {
	items := NameCards([]Offer{
		{CardName: "Winged Kuriboh", PriceAmount: "300", PriceCurrency: "CLP"},
		{CardName: "LDS3-EN100 “Winged Kuriboh” Common Effect Monster", PriceAmount: "400", PriceCurrency: "CLP"},
		{CardName: "Winged Kuriboh (PUR)", PriceAmount: "500", PriceCurrency: "CLP"},
	})
	keys := map[string]bool{}
	for _, item := range items {
		keys[item.CardKey] = true
	}
	if len(keys) != 1 || !keys["winged kuriboh"] {
		t.Fatalf("keys=%v, want one card", keys)
	}
}

// TestOnePokemonCardGroupsEditions Keeps Printings below the functional Card.
func TestOnePokemonCardGroupsEditions(t *testing.T) {
	items := NameCards([]Offer{
		{CardName: "Slowpoke", Edition: "PRE", Metadata: map[string]string{"game": "pokemon", "product_id": "610373", "functional_key": "tail-whap"}},
		{CardName: "Slowpoke", Edition: "HIF", Metadata: map[string]string{"game": "pokemon", "product_id": "197654", "functional_key": "tail-whap"}},
		{CardName: "Slowpoke", Edition: "MEG", Metadata: map[string]string{"game": "pokemon", "product_id": "704786", "functional_key": "rest"}},
	})

	if items[0].CardKey != items[1].CardKey || items[0].CardKey == items[2].CardKey {
		t.Fatalf("card keys=%q, %q, %q", items[0].CardKey, items[1].CardKey, items[2].CardKey)
	}
}

func TestUnresolvedPokemonCardKeepsItsPrinting(t *testing.T) {
	items := NameCards([]Offer{
		{CardName: "Slowpoke", Metadata: map[string]string{"game": "pokemon", "product_id": "610373"}},
		{CardName: "Slowpoke", Metadata: map[string]string{"game": "pokemon", "product_id": "197654"}},
	})

	if items[0].CardKey == items[1].CardKey {
		t.Fatalf("unresolved keys=%q, %q", items[0].CardKey, items[1].CardKey)
	}
}
