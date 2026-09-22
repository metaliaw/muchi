package application

import (
	"context"
	"net/http"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/source"
)

// A Probe Talks to the real Stores, so nothing here Throttles it. El Gate
// abierto Deja pasar cada Llamada tal como el Test la Pide.
type openGate struct{}

func (openGate) AwaitSource(context.Context, string) error { return nil }
func (openGate) RecordSource(context.Context, string, time.Duration, error) error {
	return nil
}

// probeFetcher Builds the one Client every Probe Shares.
func probeFetcher() source.Client {
	return source.Client{
		HTTP: &http.Client{Timeout: 20 * time.Second}, Gate: openGate{},
		UserAgent: userAgent, MaxAttempts: 2, BaseDelay: 250 * time.Millisecond, MaxBodyBytes: 16 << 20,
	}
}
