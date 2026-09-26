package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/errorhandler"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
)

type API struct {
	Searches           search.Service
	Health             search.HealthStore
	Inventories        InventoryShelf
	CardMetadata       map[model.Game]cardmetadata.Provider
	Autocomplete       map[model.Game]cardmetadata.AutocompleteProvider
	SupportedGames     []stores.GameSupport
	Token              string
	Logger             *slog.Logger
	HealthCheckTimeout time.Duration
}

// InventoryShelf Forgets the Published Lists a Store Keeps, so the next Search
// Reads them again instead of Waiting for their Cache to Expire.
type InventoryShelf interface {
	RefreshStoreLists(ctx context.Context, store, label string) (int, error)
}

type contextKey string

const requestIDKey contextKey = "request_id"

const (
	defaultResultLimit = 50
	maxResultLimit     = 100
)

// BuildHandler Connects each API Story to its public route.
func (a API) BuildHandler() http.Handler {
	mux := http.NewServeMux()
	a.registerHealthRoutes(mux)
	a.registerSearchRoutes(mux)
	a.registerCardRoutes(mux)
	a.registerInventoryRoutes(mux)
	return a.identifyRequest(a.recoverPanic(mux))
}

// listSupportedGames Answers which Games can be Searched, and for which Kind.
//
// `kind` Narrows the List to the Games that Answer that Question: a Caller
// Drawing a Selector for Boxes Asks for `sealed` and Draws only what Exists.
// Without it the whole List Comes back, each Game Carrying both Marks, so one
// Call is Enough to Draw a Selector that Changes Kind without Asking again.

// stockRequest Names the Offers the Caller Wants Looked at again.
type stockRequest struct {
	Offers []string `json:"offers"`
}

// checkStock Visits the Stores again for Offers this Search already Found.
//
// It Answers the Question a Comparator Cannot Answer alone: the cheapest Price
// is Worth nothing if that Card is gone. The Visit Belongs here, where each
// Store Platform already has its Adapter and its Timeout.

// checkoutRequest Names the Offers the Buyer Chose and how many of each.
type checkoutRequest struct {
	Items    []cartRequestDTO    `json:"items"`
	Shipping *shippingAddressDTO `json:"shipping,omitempty"`
}

// orderRequest Names the Offers to Place a real Order for, all at the same
// Store. Shipping is Required here where checkoutRequest Leaves it optional:
// a Quote can Skip the Store, an Order cannot.
type orderRequest struct {
	Items    []cartRequestDTO   `json:"items"`
	Shipping shippingAddressDTO `json:"shipping"`
}

// createCheckoutLinks Hands the Buyer one Way into each Store's Checkout.

type refreshInventoryReply struct {
	StoreID   string `json:"store_id"`
	List      string `json:"list,omitempty"`
	Refreshed int    `json:"refreshed"`
}

func (a API) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		value := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		sameLength := len(value) == len(a.Token)
		sameToken := subtle.ConstantTimeCompare([]byte(value), []byte(a.Token)) == 1
		valid := sameLength && sameToken
		if !valid || a.Token == "" {
			errorhandler.Handler{Logger: a.Logger}.WriteDescription(
				w, r, readRequestID(r), errorhandler.Unauthorized(), nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a API) identifyRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" || len(id) > 128 {
			id = fmt.Sprintf("req_%d", time.Now().UnixNano())
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (a API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				errorhandler.Handler{Logger: a.Logger}.WriteDescription(
					w, r, readRequestID(r), errorhandler.Internal(), value)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (a API) reportError(w http.ResponseWriter, r *http.Request, err error) {
	errorhandler.Handler{Logger: a.Logger}.Write(w, r, readRequestID(r), err)
}

func (a API) reportDescription(w http.ResponseWriter, r *http.Request, description errorhandler.Description) {
	errorhandler.Handler{Logger: a.Logger}.WriteDescription(w, r, readRequestID(r), description, nil)
}

func requireIdempotency(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" || len(key) > 128 {
		errorhandler.Handler{}.WriteDescription(
			w, r, readRequestID(r), errorhandler.MissingIdempotencyKey(), nil)
		return "", false
	}
	return key, true
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return ensureEOF(decoder)
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return errors.New("request must contain one JSON value")
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func readRequestID(r *http.Request) string {
	id, _ := r.Context().Value(requestIDKey).(string)
	return id
}
