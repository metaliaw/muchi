package search

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// SearchService Opens, Reads and Cancels Searches. A retried Create or Cancel
// with the same Key and Payload Answers the Search it already Made.
type SearchService struct {
	Repository  SearchRepository
	Tasks       TaskQueue
	MaxCards    int
	MaxQuantity int
}

func (s SearchService) CreateSearch(ctx context.Context, key string, input model.CreateInput) (model.Job, error) {
	if err := ValidateCreate(input, s.MaxCards, s.MaxQuantity); err != nil {
		return model.Job{}, err
	}
	hash := HashPayload(input)
	job, err := s.Repository.CreateSearch(ctx, key, hash, input)
	if err != nil || s.Tasks == nil {
		return job, err
	}
	return job, s.Tasks.DispatchSearch(ctx, job)
}

func (s SearchService) GetSearch(ctx context.Context, id string) (model.Job, error) {
	return s.Repository.GetSearch(ctx, id)
}

func (s SearchService) ListResults(ctx context.Context, id string, page model.ResultPage) (model.Result, error) {
	return s.Repository.ListResults(ctx, id, page)
}

func (s SearchService) CancelSearch(ctx context.Context, id, key string) (model.Job, error) {
	hash := HashPayload(map[string]string{"search_id": id, "action": "cancel"})
	return s.Repository.CancelSearch(ctx, id, key, hash)
}
