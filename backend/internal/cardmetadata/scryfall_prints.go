package cardmetadata

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

func (s Scryfall) CardPrints(ctx context.Context, name string) ([]Print, error) {
	query := url.Values{
		"q":      {fmt.Sprintf(`!%q`, name)},
		"unique": {"prints"},
	}
	body, err := s.fetch(ctx, "/cards/search", query)
	if err != nil {
		return nil, err
	}
	var result cardList
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	prints := make([]Print, 0, len(result.Data))
	for index := range result.Data {
		metadata := metadataOf(&result.Data[index])
		if metadata.Image == "" {
			continue
		}
		prints = append(prints, Print{
			Edition:         strings.ToLower(result.Data[index].Set),
			EditionName:     strings.ToLower(result.Data[index].SetName),
			CollectorNumber: strings.ToLower(result.Data[index].CollectorNumber),
			Image:           metadata.Image,
		})
	}
	return prints, nil
}
