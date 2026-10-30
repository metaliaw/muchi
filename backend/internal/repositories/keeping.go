package repositories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/search"
)

// lapsed Says that a Record's Expiry already Passed.
func lapsed(expiresAt time.Time) bool {
	return expiresAt.Before(time.Now().UTC())
}

// buildID Names a new Record: a Kind Prefix and a Random Suffix.
func buildID(prefix string) string {
	data := make([]byte, 12)
	_, _ = rand.Read(data)
	return prefix + "_" + hex.EncodeToString(data)
}

// mapSearchError Answers a Missing Record with the Search's own Failure.
func mapSearchError(err error) error {
	if errors.Is(err, ErrMissing) {
		return search.ErrNotFound
	}
	return err
}

// readRequest Reads what one Action and Key Answered before, and Says whether
// that Answer is still Alive.
func readRequest(ctx context.Context, tx Collections, action, key string) (Request, bool, error) {
	request, err := tx.Requests.Get(ctx, action+":"+key)
	if errors.Is(err, ErrMissing) {
		return Request{}, false, nil
	}
	if err != nil {
		return Request{}, false, err
	}
	return request, !lapsed(request.ExpiresAt), nil
}

// saveRequest Remembers the Answer one Action and Key Gave until Expiry.
func saveRequest(ctx context.Context, tx Collections, action, key string, request Request) error {
	return tx.Requests.Save(ctx, action+":"+key, request)
}

// readSearch Reads one Search that has not Expired; a Missing or Lapsed one is
// Answered as not Found.
func readSearch(ctx context.Context, searches Repository[Search], id string) (Search, error) {
	stored, err := searches.Get(ctx, id)
	if err != nil {
		return Search{}, mapSearchError(err)
	}
	if lapsed(stored.ExpiresAt) {
		return Search{}, search.ErrNotFound
	}
	return stored, nil
}
