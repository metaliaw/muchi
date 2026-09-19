package search

import (
	"encoding/json"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// TestAnItemKeepsItsOptionsThroughStorage Covers the Round Trip the Worker makes.
// The Item is Written as JSON and Read back by another Process: an Option that
// does not Survive that Trip is an Option the Worker never Sees.
func TestAnItemKeepsItsOptionsThroughStorage(t *testing.T) {
	stored := Item{
		ID: "item-1", SearchID: "search-1", Game: GameMagic,
		OriginalName: "Sol Ring", NormalizedName: "sol ring", Quantity: 1,
		Status: ItemPending, Offers: []offer.Offer{},
		VerifyStock: true, Match: offer.MatchIncludes,
	}
	data, err := json.Marshal(stored)
	if err != nil {
		t.Fatal(err)
	}
	var read Item
	if err := json.Unmarshal(data, &read); err != nil {
		t.Fatal(err)
	}
	if !read.VerifyStock {
		t.Error("verify_stock did not reach the worker")
	}
	if read.Match != offer.MatchIncludes {
		t.Errorf("match = %q", read.Match)
	}
}

// TestARunningItemStillCarriesItsSequence Covers what the Contract Promises.
// `sequence` is Required, and a Running Item has not Earned one yet: with
// `omitempty` the Zero Vanished and the Field went Missing from every Item
// still Working. A Client Reading the Contract Straight found no Field there.
func TestARunningItemStillCarriesItsSequence(t *testing.T) {
	running := Item{
		ID: "item-1", SearchID: "search-1", Game: GameMagic,
		OriginalName: "Sol Ring", NormalizedName: "sol ring", Quantity: 1,
		Status: ItemRunning, Offers: []offer.Offer{},
	}
	data, err := json.Marshal(running)
	if err != nil {
		t.Fatal(err)
	}
	var read map[string]any
	if err := json.Unmarshal(data, &read); err != nil {
		t.Fatal(err)
	}
	if _, found := read["sequence"]; !found {
		t.Error("sequence went missing from a running item")
	}
}
