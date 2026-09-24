package nativeapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
)

func (api *Router) addPlayerRoute(r chi.Router) {
	constructor := func(ctx context.Context) rest.Repository {
		return api.ds.Resource(ctx, model.Player{})
	}
	r.Route("/player", func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		r.Post("/", rest.Post(constructor))
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			r.Put("/", rest.Put(constructor))
			r.Delete("/", rest.Delete(constructor))
			r.Post("/apiKey", generatePlayerAPIKey(api.ds))
			r.Delete("/apiKey", revokePlayerAPIKey(api.ds))
		})
	})
}

func generatePlayerAPIKey(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key, err := ds.Player(r.Context()).GenerateAPIKey(chi.URLParam(r, "id"))
		if err != nil {
			writePlayerAPIKeyError(w, r, err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"apiKey": key})
	}
}

func revokePlayerAPIKey(ds model.DataStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := ds.Player(r.Context()).RevokeAPIKey(chi.URLParam(r, "id")); err != nil {
			writePlayerAPIKeyError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writePlayerAPIKeyError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, rest.ErrPermissionDenied):
		http.Error(w, "Forbidden", http.StatusForbidden)
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Not Found", http.StatusNotFound)
	default:
		log.Error(r.Context(), "Error changing player API key", "id", chi.URLParam(r, "id"), err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
