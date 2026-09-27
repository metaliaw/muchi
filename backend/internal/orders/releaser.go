// Package orders Reconciles a placed Order whose own Clock ran out. It backs
// the ReleaseOrders Entry-point, a Sibling of sweep.Sweeper: both Repair
// Drift on a Schedule, but neither Shape fits the other's Work.
package orders

import (
	"context"
	"log/slog"
	"time"
)

// Expirer Releases every `pending` Order past its own Cutoff, and Answers
// how many it Moved.
type Expirer interface {
	ReleaseExpiredOrders(ctx context.Context, olderThan time.Duration, limit int) (int, error)
}

// Releaser Frees the Stock a pending Order held past its own Patience: a bank
// Transfer that never Arrived must not Hold a Card forever.
type Releaser struct {
	Orders Expirer
	Logger *slog.Logger
	// MaxAge Names how long a pending Order gets before its Stock is Freed.
	MaxAge time.Duration
	// MaxReleases caps one Sweep, the way sweep.Sweeper.MaxWakes caps its own:
	// a Queue that drifted for a Week should not be Released in one Turn.
	MaxReleases int
}

// ReleaseOrders Runs one Sweep and Says how many Orders it Released.
func (r Releaser) ReleaseOrders(ctx context.Context) (int, error) {
	limit := r.MaxReleases
	if limit <= 0 {
		limit = 50
	}
	age := r.MaxAge
	if age <= 0 {
		age = 24 * time.Hour
	}
	released, err := r.Orders.ReleaseExpiredOrders(ctx, age, limit)
	if err != nil {
		return 0, err
	}
	if released > 0 {
		// Every Release here is a Transfer that never Arrived: worth Seeing,
		// not an Error a Turn should Fail over.
		r.say().Warn("Released Expired Orders", "released", released, "capped", released == limit)
	}
	return released, nil
}

func (r Releaser) say() *slog.Logger {
	if r.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return r.Logger
}
