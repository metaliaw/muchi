package jumpseller

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strconv"
	"strings"

	"github.com/cangrejometralleta/muchi-api/internal/offer"
)

// findProducts Tracks Product IDs because Different Products May Share Permalinks.
func (c Client) findProducts(ctx context.Context, name string) ([]string, error) {
	seen := map[int64]bool{}
	paths := make([]string, 0)
	for page := 1; ; page++ {
		data, err := c.fetchPage(ctx, "/api/search/"+url.PathEscape(name)+"?page="+strconv.Itoa(page))
		if err != nil {
			return nil, err
		}
		var reply struct {
			Products []struct {
				ID        int64  `json:"id"`
				Name      string `json:"name"`
				Permalink string `json:"permalink"`
			} `json:"products"`
		}
		if err := json.Unmarshal(data, &reply); err != nil {
			return nil, err
		}
		if reply.Products == nil {
			return nil, errors.New("Jumpseller response missing products")
		}
		if len(reply.Products) == 0 {
			return uniquePaths(paths), nil
		}
		added := 0
		for _, product := range reply.Products {
			if product.ID <= 0 || product.Name == "" {
				return nil, errors.New("invalid Jumpseller search product")
			}
			if seen[product.ID] {
				continue
			}
			seen[product.ID] = true
			added++
			if !offer.MatchesCard(product.Name, name) {
				continue
			}
			if product.Permalink == "" || strings.ContainsAny(product.Permalink, "/\\.") {
				return nil, errors.New("invalid Jumpseller permalink")
			}
			paths = append(paths, "/"+url.PathEscape(product.Permalink))
		}
		if added == 0 {
			return nil, errors.New("Jumpseller pagination repeated products")
		}
	}
}
