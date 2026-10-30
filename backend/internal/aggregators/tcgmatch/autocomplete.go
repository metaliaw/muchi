package tcgmatch

import (
	"context"
)

const autocompleteLimit = 3

func (c Client) Autocomplete(ctx context.Context, text, language string) ([]string, error) {
	base, err := c.validBase(text)
	if err != nil {
		return nil, err
	}
	products, err := c.searchCatalog(ctx, base, text, "card")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, autocompleteLimit)
	seen := map[string]bool{}
	for _, product := range products {
		if !c.matches(product, language) || seen[product.Name] {
			continue
		}
		names = append(names, product.Name)
		seen[product.Name] = true
		if len(names) == autocompleteLimit {
			break
		}
	}
	return names, nil
}
