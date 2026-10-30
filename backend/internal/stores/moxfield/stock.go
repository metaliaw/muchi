package moxfield

import (
	"context"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// CheckStock Asks the List again whether one of its Entries is still There.
//
// A Store that Publishes Lists has no Product Page to Visit: every Offer Points
// at the same Deck URL, so the Host tells the Checker nothing. The Answer Lives
// in the List itself, and Reading it again is the only Visit there is.
func (s Shelf) CheckStock(ctx context.Context, item model.Offer) (model.StockReading, error) {
	list := s.listOf(item)
	if list == nil {
		return model.ReadStock("unknown"), ErrNoList
	}
	return list.CheckStock(ctx, item)
}

// listOf Finds the List an Offer came from by the Id that List Wrote on it.
func (s Shelf) listOf(item model.Offer) *Client {
	for _, list := range s.Lists {
		id, err := ExtractListID(list.ListURL)
		if err != nil {
			continue
		}
		if strings.HasPrefix(item.ID, "moxfield:"+list.StoreID+":"+id+":") {
			return list
		}
	}
	return nil
}

// CheckStock Reads the List past its Cache and Counts the Entry again.
//
// Reading the Cache would Answer with the same Inventory the Search already
// Used, which Confirms nothing. The Fetch is the Point: it is the Visit other
// Stores Receive on their Product Page.
func (c *Client) CheckStock(ctx context.Context, item model.Offer) (model.StockReading, error) {
	id, err := ExtractListID(c.ListURL)
	if err != nil {
		return model.ReadStock("unknown"), err
	}
	items, err := c.reloadInventory(ctx, id)
	if err != nil {
		return model.ReadStock("unknown"), err
	}
	for _, candidate := range items {
		if candidate.ID != item.ID {
			continue
		}
		units, err := strconv.Atoi(candidate.Metadata["quantity"])
		if err != nil || units <= 0 {
			return model.CountStock("unavailable", 0), nil
		}
		return model.CountStock("available", units), nil
	}
	// An Entry the List no longer Holds is Sold: the Inventory Drops whatever
	// Counts zero, so Absence here is the Store Saying no, not Saying nothing.
	return model.CountStock("unavailable", 0), nil
}

// reloadInventory Fetches the List and Leaves it Cached for the next Search.
// The Visit is Paid once: Throwing the Result away would Make the next Search
// Pay it again.
func (c *Client) reloadInventory(ctx context.Context, id string) ([]model.Offer, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	items, err := c.fetchInventory(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Cache != nil && c.TTL > 0 {
		if err := c.Cache.SaveOffers(ctx, c.inventoryKey(id), items, c.TTL); err != nil && c.Logger != nil {
			c.Logger.WarnContext(ctx, "Inventory Cache Failed", "list", id, "error", err)
		}
	}
	return items, nil
}
