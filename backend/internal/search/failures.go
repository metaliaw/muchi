package search

import "errors"

var (
	ErrConflict   = errors.New("idempotency conflict")
	ErrInvalid    = errors.New("invalid request")
	ErrNotFound   = errors.New("search not found")
	ErrNotRunning = errors.New("search is not cancellable")

	ErrNoCardMetadata = errors.New("card metadata is not supported for this game")
	ErrNoAutocomplete = errors.New("card autocomplete is not supported for this game")
	ErrNoCardName     = errors.New("card name is required")

	ErrNoCardNames         = errors.New("card name resolution is not supported for this game")
	ErrCardNotFound        = errors.New("card was not found")
	ErrUnsupportedLanguage = errors.New("language is not supported for this game")
)
