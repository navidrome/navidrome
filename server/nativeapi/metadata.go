package nativeapi

import (
	"context"
	"errors"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/core/external"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) addMetadataRoute(r chi.Router) {
	r.Post("/metadata/{kind}/{id}/refresh", api.refreshMetadata())
}

// refreshMetadata clears the artwork state deliberately, so a wrong pick cannot be served from
// cache again; reads fall back to local resolution while the worker re-runs the chain at Bump.
func (api *Router) refreshMetadata() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kind, _ := model.ParseKind(chi.URLParam(r, "kind"))
		id := chi.URLParam(r, "id")
		if !slices.Contains(artwork.RefreshableKinds, kind) {
			http.Error(w, "invalid artwork kind", http.StatusBadRequest)
			return
		}
		if _, err := artwork.ItemName(ctx, api.ds, kind, id); err != nil {
			if errors.Is(err, model.ErrNotFound) {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}
			log.Error(ctx, "Error looking up item to refresh", "kind", kind, "id", id, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if err := artwork.Refresh(ctx, api.ds, kind, id); err != nil {
			log.Error(ctx, "Error refreshing artwork", "kind", kind, "id", id, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if external.HasInfo(kind) {
			// Detached: the request context is cancelled the moment this handler returns 204.
			bg := context.WithoutCancel(ctx)
			go func() {
				if err := api.provider.RefreshInfo(bg, kind, id); err != nil {
					log.Error(bg, "Error refreshing external info", "kind", kind, "id", id, err)
				}
			}()
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
