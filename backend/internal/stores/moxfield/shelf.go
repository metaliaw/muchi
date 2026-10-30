package moxfield

import (
	"context"
	"errors"
	"strings"
)

// ErrNoList Answers a Refresh that Named a Store or a Label nobody Publishes.
var ErrNoList = errors.New("no Moxfield list matches")

// Shelf Holds every Configured List so a Caller can Forget one on Demand.
// It Keeps the Clients the Searches Use, never a second Copy of them.
type Shelf struct {
	Lists []*Client
}

// RefreshStoreLists Drops the Cached Inventory of one Store, or of one Label
// inside it when the Caller Names one, and Returns how many Lists it Forgot.
func (s Shelf) RefreshStoreLists(ctx context.Context, store, label string) (int, error) {
	selected := s.selectLists(store, label)
	if len(selected) == 0 {
		return 0, ErrNoList
	}

	dropped := 0
	for _, list := range selected {
		if err := list.DropInventory(ctx); err != nil {
			return dropped, err
		}
		dropped++
	}

	return dropped, nil
}

// selectLists Matches a Store by its Identifier or its Name, and a Label exactly.
func (s Shelf) selectLists(store, label string) []*Client {
	selected := make([]*Client, 0, len(s.Lists))
	for _, list := range s.Lists {
		if !namesStore(list, store) {
			continue
		}
		if label != "" && !strings.EqualFold(list.Label, label) {
			continue
		}
		selected = append(selected, list)
	}
	return selected
}

func namesStore(list *Client, store string) bool {
	return strings.EqualFold(list.StoreID, store) || strings.EqualFold(list.Store, store)
}
