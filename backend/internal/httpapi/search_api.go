package httpapi

import (
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"net/http"
	"strconv"
)

func (a API) registerSearchRoutes(mux *http.ServeMux) {
	mux.Handle("GET /v1/supported-games", a.authenticate(http.HandlerFunc(a.listSupportedGames)))
	mux.Handle("POST /v1/searches", a.authenticate(http.HandlerFunc(a.createSearch)))
	mux.Handle("GET /v1/searches/{search_id}", a.authenticate(http.HandlerFunc(a.getSearch)))
	mux.Handle("GET /v1/searches/{search_id}/results", a.authenticate(http.HandlerFunc(a.listResults)))
	mux.Handle("POST /v1/searches/{search_id}/stock", a.authenticate(http.HandlerFunc(a.checkStock)))
	mux.Handle("POST /v1/searches/{search_id}/checkout", a.authenticate(http.HandlerFunc(a.createCheckoutLinks)))
	mux.Handle("POST /v1/searches/{search_id}/orders", a.authenticate(http.HandlerFunc(a.placeOrder)))
	mux.Handle("POST /v1/searches/{search_id}/cancel", a.authenticate(http.HandlerFunc(a.cancelSearch)))
}

func (a API) listSupportedGames(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind != "" && kind != string(model.KindSingle) && kind != string(model.KindSealed) {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	games := make([]stores.GameSupport, 0, len(a.SupportedGames))
	for _, game := range a.SupportedGames {
		if kind != "" && !game.Asked(kind == string(model.KindSealed)) {
			continue
		}
		games = append(games, game)
	}
	writeJSON(w, http.StatusOK, map[string]any{"games": games})
}

func (a API) createSearch(w http.ResponseWriter, r *http.Request) {
	key, ok := requireIdempotency(w, r)
	if !ok {
		return
	}
	input, err := decodeSearch(r)
	if err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	job, err := a.Searches.CreateSearch(r.Context(), key, input)
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, renderJob(job))
}

func (a API) getSearch(w http.ResponseWriter, r *http.Request) {
	job, err := a.Searches.GetSearch(r.Context(), r.PathValue("search_id"))
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, renderJob(job))
}

func (a API) listResults(w http.ResponseWriter, r *http.Request) {
	page, err := decodeResultPage(r)
	if err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	result, err := a.Searches.ListResults(r.Context(), r.PathValue("search_id"), page)
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, renderResult(result))
}

func decodeResultPage(r *http.Request) (model.ResultPage, error) {
	after, err := readQueryNumber(r, "after", 0)
	if err != nil || after < 0 {
		return model.ResultPage{}, search.ErrInvalid
	}
	limit, err := readQueryNumber(r, "limit", defaultResultLimit)
	if err != nil || limit < 1 || limit > maxResultLimit {
		return model.ResultPage{}, search.ErrInvalid
	}
	return model.ResultPage{After: after, Limit: limit}, nil
}

func readQueryNumber(r *http.Request, name string, fallback int) (int, error) {
	value := r.URL.Query().Get(name)
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func (a API) checkStock(w http.ResponseWriter, r *http.Request) {
	var request stockRequest
	if err := decodeJSON(r, &request); err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	readings, err := a.Searches.CheckOfferStock(r.Context(), r.PathValue("search_id"), request.Offers)
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"offers": renderOfferStockList(readings)})
}

func (a API) createCheckoutLinks(w http.ResponseWriter, r *http.Request) {
	var request checkoutRequest
	if err := decodeJSON(r, &request); err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	checkouts, err := a.Searches.CheckoutLinks(r.Context(), r.PathValue("search_id"), buildCartRequestList(request.Items), buildShippingAddressPointer(request.Shipping))
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stores": renderStoreCheckoutList(checkouts)})
}

func (a API) placeOrder(w http.ResponseWriter, r *http.Request) {
	key, ok := requireIdempotency(w, r)
	if !ok {
		return
	}
	var request orderRequest
	if err := decodeJSON(r, &request); err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	order, err := a.Searches.PlaceOrder(r.Context(), r.PathValue("search_id"), key, buildCartRequestList(request.Items), buildShippingAddress(request.Shipping))
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, renderOrder(order))
}

func (a API) cancelSearch(w http.ResponseWriter, r *http.Request) {
	key, ok := requireIdempotency(w, r)
	if !ok {
		return
	}
	job, err := a.Searches.CancelSearch(r.Context(), r.PathValue("search_id"), key)
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, renderJob(job))
}
