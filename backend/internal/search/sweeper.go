package search

import (
	"context"
	"log/slog"
)

// WaitingCounter Says how much Work is ready for a Turn.
type WaitingCounter interface {
	CountWaitingItems(ctx context.Context, limit int) (int, error)
}

// Waker Adds Turns that belong to no Position.
type Waker interface {
	WakeWorker(ctx context.Context, reason string) error
}

// Sweeper Puts back the Wake-ups that silent Turns spent.
//
// A Task carries no Item, so Work and Wake-ups are bound only by their Count.
// A Turn that ends without working —a dead Document at the Head, a Worker that
// died mid-Turn— spends one and nothing replaces it. Left alone, Items wait
// forever with no Task coming for them, and the Front shows a Search queued
// that no one will ever pick up.
type Sweeper struct {
	Items  WaitingCounter
	Queue  Waker
	Logger *slog.Logger
	// MaxWakes caps one Sweep. A Sweeper that fires hundreds of Turns at once
	// would beat the Sources harder than any Search ever does.
	MaxWakes int
}

// SweepQueue Wakes the Worker once for each waiting Item, and Says so.
func (s Sweeper) SweepQueue(ctx context.Context) (int, error) {
	limit := s.MaxWakes
	if limit <= 0 {
		limit = 1
	}
	waiting, err := s.Items.CountWaitingItems(ctx, limit)
	if err != nil {
		return 0, err
	}
	if waiting == 0 {
		return 0, nil
	}
	for wake := range waiting {
		if err := s.Queue.WakeWorker(ctx, "sweep"); err != nil {
			s.say().Error("Sweep Failed", "woken", wake, "waiting", waiting, "error", err)
			return wake, err
		}
	}
	// Every Sweep that finds Work is a Warning: the Queue should have emptied
	// on its own. The Number says how badly the Accounting drifted.
	s.say().Warn("Sweep Replaced Lost Wake-ups", "woken", waiting, "capped", waiting == limit)
	return waiting, nil
}

func (s Sweeper) say() *slog.Logger {
	if s.Logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return s.Logger
}
