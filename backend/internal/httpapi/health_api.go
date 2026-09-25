package httpapi

import (
	"context"
	"net/http"
)

func (a API) registerHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/health", a.getHealth)
	mux.Handle("GET /v1/health/sources", a.authenticate(http.HandlerFunc(a.listSourceHealth)))
}

func (a API) getHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), a.HealthCheckTimeout)
	defer cancel()
	if err := a.Health.CheckHealth(ctx); err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a API) listSourceHealth(w http.ResponseWriter, r *http.Request) {
	items, err := a.Health.ListSourceHealth(r.Context())
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sources": renderSourceHealthList(items)})
}
