package moxfield

import (
	"context"
	"testing"
)

func buildShelf() Shelf {
	return Shelf{Lists: []*Client{
		{Cache: &inventoryCache{}, StoreID: "el-wombat-rabioso-tcg", Store: "El Wombat Rabioso TCG", Label: "Negro", ListURL: "https://moxfield.com/decks/r30AI4hXKk-xhA9U6pQQ6g", Rate: 700},
		{Cache: &inventoryCache{}, StoreID: "el-wombat-rabioso-tcg", Store: "El Wombat Rabioso TCG", Label: "Blanco", ListURL: "https://moxfield.com/decks/qWSpmofE70iJv8zpwjBXrA", Rate: 700},
	}}
}

func TestRefreshEveryListOfOneStore(t *testing.T) {
	dropped, err := buildShelf().RefreshStoreLists(context.Background(), "el-wombat-rabioso-tcg", "")
	if err != nil || dropped != 2 {
		t.Fatalf("dropped=%d err=%v", dropped, err)
	}
}

func TestRefreshOneNamedList(t *testing.T) {
	dropped, err := buildShelf().RefreshStoreLists(context.Background(), "El Wombat Rabioso TCG", "negro")
	if err != nil || dropped != 1 {
		t.Fatalf("dropped=%d err=%v", dropped, err)
	}
}

func TestRefreshUnknownStoreAnswersNoList(t *testing.T) {
	if _, err := buildShelf().RefreshStoreLists(context.Background(), "la-cripta", ""); err != ErrNoList {
		t.Fatalf("err=%v", err)
	}
}

func TestRefreshUnknownLabelAnswersNoList(t *testing.T) {
	if _, err := buildShelf().RefreshStoreLists(context.Background(), "el-wombat-rabioso-tcg", "Dorado"); err != ErrNoList {
		t.Fatalf("err=%v", err)
	}
}

func TestRefreshForgetsTheCachedList(t *testing.T) {
	cache := &inventoryCache{}
	client := &Client{Cache: cache, StoreID: "wombat", Store: "El Wombat", Label: "Negro", ListURL: "https://moxfield.com/decks/r30AI4hXKk-xhA9U6pQQ6g", Rate: 700}
	cache.key, cache.found = client.inventoryKey("r30AI4hXKk-xhA9U6pQQ6g"), true
	if _, err := (Shelf{Lists: []*Client{client}}).RefreshStoreLists(context.Background(), "wombat", ""); err != nil {
		t.Fatal(err)
	}
	if cache.found {
		t.Fatal("the cached list survived the refresh")
	}
}
