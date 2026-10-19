package repositories

import (
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// Search Keeps a Job beside the Ids of the Items it Splits into and the
// Moment it Expires.
type Search struct {
	Job       model.Job
	ItemIDs   []string
	ExpiresAt time.Time
}

// Item Keeps a card Item beside the Moment it becomes Claimable and the Moment
// it Expires. Damaged Marks a Record that Storage could not Read back: the
// Queue Discards it, never Stalls behind it.
type Item struct {
	model.Item
	AvailableAt time.Time
	ExpiresAt   time.Time
	Damaged     bool
}

// Entry Keeps a List of Offers until its Expiry. Indexed Says that every Offer
// in it also has its own Snapshot.
type Entry struct {
	Offers    []model.Offer
	ExpiresAt time.Time
	Indexed   bool
}

// Snapshot Keeps one Offer until the Entry that Carried it Expires.
type Snapshot struct {
	Offer     model.Offer
	ExpiresAt time.Time
}

// Request Remembers what a Key Answered: the Payload Hash it Came with and the
// Resource it Made.
type Request struct {
	Hash       string
	ResourceID string
	ExpiresAt  time.Time
}

// Lease Keeps the Moment one Domain may be Asked again.
type Lease struct {
	NextAllowedAt time.Time
}

// Ping Stands for the one Record a Health Check Reads.
type Ping struct{}
