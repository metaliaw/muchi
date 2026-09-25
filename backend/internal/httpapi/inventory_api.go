package httpapi

import (
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"net/http"
	"strings"
)

func (a API) registerInventoryRoutes(mux *http.ServeMux) {
	mux.Handle("POST /v1/stores/{store_id}/inventory/refresh", a.authenticate(http.HandlerFunc(a.refreshStoreInventory)))
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
