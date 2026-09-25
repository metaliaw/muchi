package search

import "errors"

var (
	ErrConflict   = errors.New("idempotency conflict")
	ErrInvalid    = errors.New("invalid request")
	ErrNotFound   = errors.New("search not found")
	ErrNotRunning = errors.New("search is not cancellable")
)
