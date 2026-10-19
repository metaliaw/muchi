package model

import "time"

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
	Name     string
	Quantity int
}

type Game string

const (
	GameDigimon   Game = "digimon"
	GameMagic     Game = "magic"
	GameMitos     Game = "mitos-y-leyendas"
	GameOnePiece  Game = "one-piece"
	GamePokemon   Game = "pokemon"
	GameRiftbound Game = "riftbound"
	GameYuGiOh    Game = "yugioh"
)

type Options struct {
	VerifyStock bool
	Match       MatchMode
	Kind        ProductKind
}

// SourceFault Names a Source that Failed while others Answered. Sin esto una
// Respuesta incompleta se Ve igual que una completa.
type SourceFault struct {
	Source string
	Reason string
}

type CreateInput struct {
	Game    Game
	Cards   []CardInput
	Options Options
}

type Job struct {
	ID          string
	Game        Game
	Status      JobStatus
	Total       int
	Processed   int
	CurrentCard string
	Found       int
	NotFound    int
	Errors      int
	CreatedAt   time.Time
	StartedAt   *time.Time
	UpdatedAt   time.Time
	FinishedAt  *time.Time
	IncidentID  string
}

type Item struct {
	ID             string
	SearchID       string
	Game           Game
	Position       int
	Sequence       int
	OriginalName   string
	NormalizedName string
	Quantity       int
	Status         ItemStatus
	Attempts       int
	Source         string
	ErrorCode      string
	ErrorMessage   string
	Offers         []Offer
	LeaseOwner     string
	LeaseUntil     *time.Time
	VerifyStock    bool
	Match          MatchMode
	Kind           ProductKind
	Faults         []SourceFault
}

type Result struct {
	SearchID string
	Items    []Item
	Cursor   int
	HasMore  bool
}

type ResultPage struct {
	After int
	Limit int
}

type SourceHealth struct {
	Source                        string
	Platform                      string
	Enabled                       bool
	EstimatedResponseMilliseconds int64
	LastSuccess                   *time.Time
	LastFailure                   *time.Time
	ConsecutiveFailures           int
	LatencyMilliseconds           int64
	CircuitOpenUntil              *time.Time
}
