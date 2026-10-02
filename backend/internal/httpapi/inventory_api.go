package httpapi

import (
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"github.com/cangrejometralleta/muchi-api/internal/stores"
	"net/http"
	"strings"
)

func (a API) registerInventoryRoutes(mux *http.ServeMux) {
	mux.Handle("GET /v1/stores", a.authenticate(http.HandlerFunc(a.listStores)))
	mux.Handle("POST /v1/stores/{store_id}/inventory/refresh", a.authenticate(http.HandlerFunc(a.refreshStoreInventory)))
}

// listStores Answers which Stores Feed the Offers and where each one Stands.
func (a API) listStores(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"stores": append([]stores.ListedStore{}, a.Stores...)})
}

func (a API) refreshStoreInventory(w http.ResponseWriter, r *http.Request) {
	if a.Inventories == nil {
		a.reportError(w, r, search.ErrNotFound)
		return
	}
	store, list := r.PathValue("store_id"), strings.TrimSpace(r.URL.Query().Get("list"))
	refreshed, err := a.Inventories.RefreshStoreLists(r.Context(), store, list)
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, refreshInventoryReply{StoreID: store, List: list, Refreshed: refreshed})
}
