package tcgmatch

import (
	"context"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
)

func (c Client) CardMetadata(ctx context.Context, request cardmetadata.Request) (cardmetadata.Metadata, error) {
	base, err := c.validBase(request.Name)
	if err != nil {
		return cardmetadata.Metadata{}, err
	}
	products, err := c.searchCatalog(ctx, base, request.Name, "card")
	if err != nil {
		return cardmetadata.Metadata{}, err
	}
	for _, product := range products {
		if !c.matches(product, request.Language) || request.Edition != "" && !strings.EqualFold(request.Edition, product.SetCode) && !strings.EqualFold(request.Edition, product.SetID) {
			continue
		}
		edition := product.SetCode
		if edition == "" {
			edition = product.SetID
		}
		return cardmetadata.Metadata{Name: product.Name, Edition: edition, Image: product.Image}, nil
	}
	return cardmetadata.Metadata{}, nil
}
