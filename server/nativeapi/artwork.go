package nativeapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

// refreshableArtworkKinds are the entity kinds a manual re-resolve accepts.
var refreshableArtworkKinds = map[string]bool{
	model.KindAlbumArtwork.Prefix():     true,
	model.KindArtistArtwork.Prefix():    true,
	model.KindPlaylistArtwork.Prefix():  true,
	model.KindRadioArtwork.Prefix():     true,
	model.KindMediaFileArtwork.Prefix(): true,
}

func (api *Router) addArtworkRoute(r chi.Router) {
	r.Post("/artwork/{kind}/{id}/refresh", api.refreshArtwork())
}

// refreshArtwork clears an item's resolved artwork state and re-queues it at Bump priority.
// State is deliberately cleared so a wrong pick disappears immediately (placeholder until re-resolved).
func (api *Router) refreshArtwork() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		kind := chi.URLParam(r, "kind")
		id := chi.URLParam(r, "id")
		if !refreshableArtworkKinds[kind] {
			http.Error(w, "invalid artwork kind", http.StatusBadRequest)
			return
		}
		if err := api.ds.Artwork(ctx).DeleteForItem(kind, id); err != nil {
			log.Error(ctx, "Error clearing artwork state", "kind", kind, "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		item := model.ArtworkQueueItem{ItemKind: kind, ItemID: id, ImageType: model.ImageTypePrimary,
			Priority: model.ArtworkPriorityBump}
		if err := api.ds.ArtworkQueue(ctx).Enqueue(item); err != nil {
			log.Error(ctx, "Error enqueuing artwork refresh", "kind", kind, "id", id, err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
