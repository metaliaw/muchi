package repositories

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/db"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/search/repositorytest"
)

func TestSearchItemRepositoryContract(t *testing.T) {
	project := os.Getenv("FIRESTORE_PROJECT_ID")
	if project == "" {
		t.Skip("set FIRESTORE_PROJECT_ID to run the database-backed contract")
	}
	store, err := db.OpenStore(context.Background(), project, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.CloseStore() })
	repository := New(store)
	input := model.CreateInput{Cards: []model.CardInput{
		{Name: "Repository contract one", Quantity: 1},
		{Name: "Repository contract two", Quantity: 1},
	}}
	key := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := repository.CreateSearch(context.Background(), key, search.HashPayload(input), input); err != nil {
		t.Fatal(err)
	}
	repositorytest.RunItemLeaseContract(t, func(*testing.T) search.SearchItemRepository { return repository })
}
