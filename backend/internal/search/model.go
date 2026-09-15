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
	VerifyStock bool            `json:"verify_stock"`
	Match       offer.MatchMode `json:"match,omitempty"`
}

// SourceFault Names a Source that Failed while others Answered. Sin esto una
// Respuesta incompleta se Ve igual que una completa.
type SourceFault struct {
	Source string `json:"source"`
	Reason string `json:"reason"`
}

type CreateInput struct {
	Game    Game        `json:"game"`
	Cards   []CardInput `json:"cards"`
	Options Options     `json:"options"`
}

type Job struct {
	ID          string     `json:"id"`
	Game        Game       `json:"game,omitempty"`
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
	// El Arriendo lo Guarda el Registro de Firestore, no el Payload: Cambia en
	// cada Reclamo y no Viaja con el Item.
	LeaseOwner string     `json:"-"`
	LeaseUntil *time.Time `json:"-"`
	// Las Opciones Viajan Serializadas porque el Worker Lee el Item de vuelta
	// desde el Almacén, en otro Proceso. Con `json:"-"` se Perdían al Escribir
	// y el Worker las Leía siempre en falso.
	VerifyStock bool            `json:"verify_stock,omitempty"`
	Match       offer.MatchMode `json:"match,omitempty"`
	// Faults Name the Sources that Fell while others Answered. Sin ellos un
	// Resultado incompleto Llega Marcado `found` y nadie lo Nota.
	Faults []SourceFault `json:"faults,omitempty"`
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
	Source                        string     `json:"source"`
	Platform                      string     `json:"platform,omitempty"`
	Enabled                       bool       `json:"enabled"`
	EstimatedResponseMilliseconds int64      `json:"estimated_response_ms"`
	LastSuccess                   *time.Time `json:"last_success,omitempty"`
	LastFailure                   *time.Time `json:"last_failure,omitempty"`
	ConsecutiveFailures           int        `json:"consecutive_failures"`
	LatencyMilliseconds           int64      `json:"latency_ms"`
	CircuitOpenUntil              *time.Time `json:"circuit_open_until,omitempty"`
}
