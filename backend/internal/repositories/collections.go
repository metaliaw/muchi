package repositories

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// Collections Names every Collection the Stores Read through the Primitive.
// A Store Opens them, and inside a Transaction the same Names are Bound to it.
type Collections struct {
	Searches     Repository[Search]
	Items        Repository[Item]
	ItemOffers   Repository[Entry]
	Requests     Repository[Request]
	Orders       Repository[model.Order]
	Offers       Repository[Snapshot]
	OfferCache   Repository[Entry]
	SourceHealth Repository[model.SourceHealth]
	Traffic      Repository[Lease]
	Health       Repository[Ping]
}

// Database Opens the Collections and Runs Work across them in one Transaction:
// every Read in work Comes before its first Write, and a Failure Undoes it all.
type Database interface {
	Collections() Collections
	Transact(ctx context.Context, work func(ctx context.Context, tx Collections) error) error
	CloseStore() error
}
