package handler

import (
	"github.com/cangrejometralleta/muchi-api/internal/cardmetadata"
	"github.com/cangrejometralleta/muchi-api/internal/model"
	"github.com/cangrejometralleta/muchi-api/internal/search"
	"net/http"
)

func (a Handler) registerCardRoutes(mux *http.ServeMux) {
	mux.Handle("GET /v1/cards/metadata", a.authenticate(http.HandlerFunc(a.getCardMetadata)))
	mux.Handle("GET /v1/cards/autocomplete", a.authenticate(http.HandlerFunc(a.autocompleteCards)))
	mux.Handle("GET /v1/cards/canonical-name", a.authenticate(http.HandlerFunc(a.resolveCardName)))
	mux.Handle("GET /v1/cards/offers", a.authenticate(http.HandlerFunc(a.findCardOffers)))
}

func (a Handler) getCardMetadata(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	metadata, err := a.Searches.ReadCardMetadata(r.Context(), readQueryGame(r), cardmetadata.Request{
		Name: query.Get("name"), Language: query.Get("language"),
		Edition: query.Get("edition"), Foil: query.Get("foil") == "true",
	})
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, metadata)
}

func (a Handler) autocompleteCards(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	names, err := a.Searches.SuggestCardNames(r.Context(), readQueryGame(r), query.Get("name"), query.Get("language"))
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"suggestions": names})
}

func (a Handler) resolveCardName(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	canonical, err := a.Searches.ResolveCardName(r.Context(), readQueryGame(r), query.Get("name"), query.Get("language"))
	if err != nil {
		a.reportError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"canonical_name": canonical})
}

// readQueryGame Reads the Game a Request Asks about; Magic when it Names none.
func readQueryGame(r *http.Request) model.Game {
	game := model.Game(r.URL.Query().Get("game"))
	if game == "" {
		return model.GameMagic
	}
	return game
}

func (a Handler) findCardOffers(w http.ResponseWriter, r *http.Request) {
	game, query, err := decodeOfferQuery(r)
	if err != nil {
		a.reportError(w, r, search.ErrInvalid)
		return
	}
	items, faults, err := a.Searches.FindCardOffers(r.Context(), game, query)
	// Every Source Falling is an Answer, not a Broken Server. The Faults Name
	// which ones Fell and Why, and the Contract already Says a non-empty List
	// Means the Offers are Incomplete. A 500 Threw that List away and Told the
	// Caller nothing it could Act on — and it Arrived as a Blank Failure right
	// next to Sources the Caller can See are Resting.
	if err != nil && len(faults) == 0 {
		a.reportError(w, r, err)
		return
	}
	// `faults` Viaja aunque Venga vacío: un Campo que Aparece sólo cuando hay
	// Problemas Enseña a no Mirarlo.
	if faults == nil {
		faults = []model.SourceFault{}
	}
	// `offers` Viaja como Arreglo aunque no haya ninguna. Cuando todas las
	// Fuentes Caen no Queda Lista que Devolver, y una Lista ausente se Serializa
	// como `null`: el Contrato Promete un Arreglo, y un Cliente que lo Recorre
	// sin Mirar Revienta justo en la Respuesta que ya Traía malas Noticias.
	if items == nil {
		items = []model.Offer{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": query.Name, "match": query.Match, "kind": query.Kind, "offers": renderOfferList(items), "faults": renderSourceFaultList(faults),
	})
}
