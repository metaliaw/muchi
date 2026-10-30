package search

import (
	"context"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

// collectSearchOffers Reads every Page of a Search and Indexes its Offers by Id.
func collectSearchOffers(ctx context.Context, repository SearchRepository, id string) (map[string]model.Offer, error) {
	found := map[string]model.Offer{}
	for cursor := 0; ; {
		page, err := repository.ListResults(ctx, id, model.ResultPage{After: cursor, Limit: maxResultPage})
		if err != nil {
			return nil, err
		}
		for _, item := range page.Items {
			for _, candidate := range item.Offers {
				found[candidate.ID] = candidate
			}
		}
		if !page.HasMore || page.Cursor <= cursor {
			return found, nil
		}
		cursor = page.Cursor
	}
}
