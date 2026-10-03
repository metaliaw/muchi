package tcgmatch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/model"
)

const catalogLimit = 24

func (c Client) searchCatalog(ctx context.Context, base *url.URL, name, productType string) ([]catalogProduct, error) {
	query := url.Values{"q": {name}, "tcg": {c.Game}, "type": {productType}, "inStock": {"true"}, "limit": {strconv.Itoa(catalogLimit)}, "page": {"1"}, "sortBy": {"listings"}}
	target := strings.TrimRight(c.BaseURL, "/") + "/catalog/search?" + query.Encode()
	data, err := c.Fetcher.FetchSource(ctx, base.Host, target)
	if err != nil {
		return nil, err
	}
	var reply catalogReply
	if err := json.Unmarshal(data, &reply); err != nil {
		return nil, fmt.Errorf("decode TCGMatch catalog: %w", err)
	}
	return reply.Products, nil
}

func (c Client) matches(product catalogProduct, language string) bool {
	if product.TCG != c.Game || product.Type != "card" {
		return false
	}
	if language == "" {
		return true
	}
	for _, available := range product.Languages {
		if strings.EqualFold(available, language) {
			return true
		}
	}
	return false
}

// catalogType Names the Catalog Type TCGMatch Files this Kind under. The Filter
// Never Leaves: an Accessory Answers `tcg` as a List, and this Reader Wants a
// String — Asking for everything Would Break the Decode.
func catalogType(kind model.ProductKind) string {
	if kind == model.KindSealed {
		return "sealed"
	}
	return "card"
}
