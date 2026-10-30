package repositories

import (
	"log/slog"
	"time"
)

// SearchOffer Serves every Search Port: it is the Stores Joined, and their
// Methods are its own. The Database Opens the Collections and Runs the
// Transactions, so the Logic here Names no Vendor.
type SearchOffer struct {
	*SearchStore
	*ItemStore
	*OrderStore
	*OfferStore
	*SourceStore
	database Database
}

// New Joins the Stores over one Database, Keeping Searches and Orders for ttl.
func New(database Database, ttl time.Duration) *SearchOffer {
	return &SearchOffer{
		SearchStore: NewSearchStore(database, ttl),
		ItemStore:   NewItemStore(database),
		OrderStore:  NewOrderStore(database, ttl),
		OfferStore:  NewOfferStore(database),
		SourceStore: NewSourceStore(database),
		database:    database,
	}
}

// TellStore Gives the Queue a Voice, since it is the Queue that Warns.
func (r *SearchOffer) TellStore(logger *slog.Logger) {
	r.ItemStore.TellStore(logger)
}

func (r *SearchOffer) Close() error { return r.database.CloseStore() }
