// Package catalog Turns the Store and Provider Config into the Sources a Search
// asks. A Builder Reads stores.yaml through the Shops and Builds Providers,
// SourcesByGame, print and set libraries, and the product-kind limits; the
// Application Composes the Runtime from those Parts.
package catalog

import (
	"log/slog"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/stores/commerce"
	"github.com/cangrejometralleta/muchi-api/internal/stores/moxfield"
)

// Builder Holds what every Part Needs to be Built: the Shops, which Carry the
// Config and the Fetcher, and what a Moxfield List Needs to Cache its Prices.
type Builder struct {
	Shops  commerce.Shops
	Cache  moxfield.InventoryCache
	TTL    time.Duration
	Logger *slog.Logger
}
