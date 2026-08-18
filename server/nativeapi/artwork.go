package nativeapi

import (
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) addArtworkRoute(r chi.Router) {
	r.Post("/artwork/{kind}/{id}/refresh", api.refreshArtwork())
}

// State is deliberately cleared so a wrong pick disappears immediately (placeholder until re-resolved).
func (api *Router) refreshArtwork() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kind, _ := model.ParseKind(chi.URLParam(r, "kind"))
		id := chi.URLParam(r, "id")
		if !slices.Contains(artwork.RefreshableKinds, kind) {
			http.Error(w, "invalid artwork kind", http.StatusBadRequest)
			return
		}
		if err := artwork.Refresh(ctx, api.ds, kind, id); err != nil {
			log.Error(ctx, "Error refreshing artwork", "kind", kind, "id", id, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
