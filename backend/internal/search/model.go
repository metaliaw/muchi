package search

import (
	"errors"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

var (
	ErrConflict   = errors.New("idempotency conflict")
	ErrInvalid    = errors.New("invalid request")
	ErrNotFound   = errors.New("search not found")
	ErrNotRunning = errors.New("search is not cancellable")
)

type JobStatus string

const (
	JobQueued              JobStatus = "queued"
	JobRunning             JobStatus = "running"
	JobCompleted           JobStatus = "completed"
	JobCompletedWithErrors JobStatus = "completed_with_errors"
	JobFailed              JobStatus = "failed"
	JobCancelled           JobStatus = "cancelled"
)

type ItemStatus string

const (
	ItemPending     ItemStatus = "pending"
	ItemRunning     ItemStatus = "running"
	ItemFound       ItemStatus = "found"
	ItemNotFound    ItemStatus = "not_found"
	ItemSourceError ItemStatus = "source_error"
)

type CardInput struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
}

type Game string

const (
	GameMagic    Game = "magic"
	GameOnePiece Game = "one-piece"
	GamePokemon  Game = "pokemon"
	GameYuGiOh   Game = "yugioh"
)

type Options struct {
	VerifyStock bool `json:"verify_stock"`
	StoresOnly  bool `json:"stores_only"`
}

type CreateInput struct {
	Game    Game        `json:"game"`
	Cards   []CardInput `json:"cards"`
	Options Options     `json:"options"`
}

type Job struct {
	ID          string     `json:"id"`
	Status      JobStatus  `json:"status"`
	Total       int        `json:"total"`
	Processed   int        `json:"processed"`
	CurrentCard string     `json:"current_card,omitempty"`
	Found       int        `json:"found"`
	NotFound    int        `json:"not_found"`
	Errors      int        `json:"errors"`
	CreatedAt   time.Time  `json:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	IncidentID  string     `json:"incident_id,omitempty"`
}

type Item struct {
	ID             string        `json:"id"`
	SearchID       string        `json:"search_id"`
	Game           Game          `json:"game"`
	Position       int           `json:"position"`
	Sequence       int           `json:"sequence,omitempty"`
	OriginalName   string        `json:"original_name"`
	NormalizedName string        `json:"normalized_name"`
	Quantity       int           `json:"quantity"`
	Status         ItemStatus    `json:"status"`
	Attempts       int           `json:"attempts"`
	Source         string        `json:"source,omitempty"`
	ErrorCode      string        `json:"error_code,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	Offers         []offer.Offer `json:"offers"`
	LeaseOwner     string        `json:"-"`
	LeaseUntil     *time.Time    `json:"-"`
	VerifyStock    bool          `json:"-"`
	StoresOnly     bool          `json:"-"`
}

type Result struct {
	SearchID string `json:"search_id"`
	Items    []Item `json:"items"`
	Cursor   int    `json:"cursor"`
	HasMore  bool   `json:"has_more"`
}

type ResultPage struct {
	After int
	Limit int
}

type SourceHealth struct {
	Source              string     `json:"source"`
	LastSuccess         *time.Time `json:"last_success,omitempty"`
	LastFailure         *time.Time `json:"last_failure,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	LatencyMilliseconds int64      `json:"latency_ms"`
	CircuitOpenUntil    *time.Time `json:"circuit_open_until,omitempty"`
}
