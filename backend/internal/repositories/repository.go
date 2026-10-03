// Package repositories Holds the one primitive every Collection speaks, and the
// Stores that Build the Ports on top of it.
package repositories

import (
	"context"
	"errors"
)

// ErrMissing Answers an Id the Collection never Kept.
var ErrMissing = errors.New("record missing")

// ErrExists Answers a Create over an Id the Collection already Keeps.
var ErrExists = errors.New("record already exists")

// Repository Keeps one Kind of Record by Id. Create Refuses an Id already
// Kept and Save Writes over it. Update Reads, Changes and Writes in one
// Transaction: an Error from change Aborts it and Comes back untouched.
// GetMany Answers only the Records it Found, and SaveMany Writes in Bulk.
type Repository[T any] interface {
	Create(ctx context.Context, id string, value T) error
	Save(ctx context.Context, id string, value T) error
	Get(ctx context.Context, id string) (T, error)
	Update(ctx context.Context, id string, change func(T) (T, error)) (T, error)
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, query Query) ([]T, error)
	GetMany(ctx context.Context, ids []string) ([]Keyed[T], error)
	SaveMany(ctx context.Context, records []Keyed[T]) error
}

// Keyed Carries a Record with the Id it is Kept under.
type Keyed[T any] struct {
	ID    string
	Value T
}

// Operator Compares a Field with a Value.
type Operator string

const (
	Equal   Operator = "=="
	AtMost  Operator = "<="
	AtLeast Operator = ">="
	Greater Operator = ">"
)

// Filter Keeps the Records whose Field Compares with Value.
type Filter struct {
	Field    string
	Operator Operator
	Value    any
}

// Query Narrows a List: every Filter Holds, the Order Sorts by one Field
// Ascending, and a Limit of zero Asks for all.
type Query struct {
	Filters []Filter
	OrderBy string
	Limit   int
}
