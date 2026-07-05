package jellyfin

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

func (api *Router) getItemImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemId := chi.URLParam(r, "itemId")
	size, _ := strconv.Atoi(r.URL.Query().Get("maxWidth"))

	artID := api.resolveArtworkID(ctx, itemId)
	reader, _, err := api.artwork.GetOrPlaceholder(ctx, artID, size, false)
	switch {
	case errors.Is(err, context.Canceled):
		return
	case err != nil:
		log.Warn(ctx, "Error retrieving artwork", "id", itemId, err)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	defer reader.Close()
	// Content-Type is left unset so net/http sniffs it from the first bytes written
	// (placeholders are WebP; unresized covers can be PNG/WebP/etc, not always JPEG).
	_, _ = io.Copy(w, reader)
}

// resolveArtworkID maps a bare Jellyfin item id to a Navidrome ArtworkID string
// by probing album -> artist -> media file (music items only).
func (api *Router) resolveArtworkID(ctx context.Context, itemId string) string {
	if al, err := api.ds.Album(ctx).Get(itemId); err == nil {
		return al.CoverArtID().String()
	}
	if ar, err := api.ds.Artist(ctx).Get(itemId); err == nil {
		return ar.CoverArtID().String()
	}
	if mf, err := api.ds.MediaFile(ctx).Get(itemId); err == nil {
		return mf.CoverArtID().String()
	}
	return (model.ArtworkID{}).String()
}
