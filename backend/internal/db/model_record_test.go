package db

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// TestAnItemKeepsItsOptionsThroughStorage Covers the Round Trip the Worker makes.
// The Item is Written as JSON and Read back by another Process: an Option that
// does not Survive that Trip is an Option the Worker never Sees.
func TestAnItemKeepsItsOptionsThroughStorage(t *testing.T) {
	stored := model.Item{
		ID: "item-1", SearchID: "search-1", Game: model.GameMagic,
		OriginalName: "Sol Ring", NormalizedName: "sol ring", Quantity: 1,
		Status: model.ItemPending, Offers: []model.Offer{},
		VerifyStock: true, Match: model.MatchIncludes,
	}
	data, err := json.Marshal(renderItem(stored))
	if err != nil {
		t.Fatal(err)
	}
	var payload itemPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	read := buildItem(payload)
	if !read.VerifyStock {
		t.Error("verify_stock did not reach the worker")
	}
	if read.Match != model.MatchIncludes {
		t.Errorf("match = %q", read.Match)
	}
}

// TestARunningItemStillCarriesItsSequence Covers what the Contract Promises.
// `sequence` is Required, and a Running Item has not Earned one yet: with
// `omitempty` the Zero Vanished and the Field went Missing from every Item
// still Working. A Client Reading the Contract Straight found no Field there.
func TestARunningItemStillCarriesItsSequence(t *testing.T) {
	running := model.Item{
		ID: "item-1", SearchID: "search-1", Game: model.GameMagic,
		OriginalName: "Sol Ring", NormalizedName: "sol ring", Quantity: 1,
		Status: model.ItemRunning, Offers: []model.Offer{},
	}
	data, err := json.Marshal(renderItem(running))
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

func TestExistingItemPayloadReachesWorker(t *testing.T) {
	const payload = `{"id":"item-1","search_id":"search-1","game":"magic","position":2,"sequence":0,"original_name":"Sol Ring","normalized_name":"sol ring","quantity":2,"status":"found","attempts":1,"verify_stock":true,"match":"includes","kind":"sealed","offers":[{"id":"offer-1","card_name":"Sol Ring","store":"Shop","price_amount":"1000","price_currency":"CLP","url":"https://shop.test/card","variant_id":"123","stock_status":"available","stock_quantity":2,"suspicious":false,"metadata":{"product_id":"123"}}],"faults":[{"source":"other.test","reason":"timeout"}]}`
	var stored itemPayload
	if err := json.Unmarshal([]byte(payload), &stored); err != nil {
		t.Fatal(err)
	}
	item := buildItem(stored)
	if item.ID != "item-1" || item.SearchID != "search-1" || item.Game != model.GameMagic || item.Quantity != 2 || item.Position != 2 || !item.VerifyStock || item.Match != model.MatchIncludes || item.Kind != model.KindSealed {
		t.Fatalf("worker item = %#v", item)
	}
	if len(item.Offers) != 1 {
		t.Fatalf("offers = %#v", item.Offers)
	}
	offer := item.Offers[0]
	if offer.CardName != "Sol Ring" || offer.PriceAmount != "1000" || offer.VariantID != "123" || offer.StockQuantity == nil || *offer.StockQuantity != 2 || offer.Metadata["product_id"] != "123" {
		t.Fatalf("offer = %#v", offer)
	}
	if len(item.Faults) != 1 || item.Faults[0].Source != "other.test" {
		t.Fatalf("faults = %#v", item.Faults)
	}
	item.LeaseOwner = "worker-private"
	data, err := json.Marshal(renderItem(item))
	if err != nil {
		t.Fatal(err)
	}
	var encoded map[string]any
	if err := json.Unmarshal(data, &encoded); err != nil {
		t.Fatal(err)
	}
	if encoded["verify_stock"] != true || encoded["search_id"] != "search-1" || encoded["sequence"] != float64(0) {
		t.Fatalf("stored payload = %s", data)
	}
	if strings.Contains(string(data), "worker-private") {
		t.Fatalf("lease leaked into payload: %s", data)
	}
}
