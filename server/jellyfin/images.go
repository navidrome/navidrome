package jellyfin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
)

func (api *Router) getItemImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	itemId := dto.DecodeID(chi.URLParam(r, "itemId"))
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
// by probing album -> artist -> media file -> playlist.
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
	if pl, err := api.ds.Playlist(ctx).Get(itemId); err == nil {
		return pl.CoverArtID().String()
	}
	return (model.ArtworkID{}).String()
}

// postItemImage handles cover image upload. Only playlists support upload through this API;
// album/artist covers come from tag/sidecar scanning. The body is always drained first —
// even on the not-implemented path — because Jellyfin clients (e.g. Finamp) write it
// synchronously and see a broken pipe if we respond before reading it.
func (api *Router) postItemImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "itemId"))

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if _, err := api.playlists.Get(ctx, id); err != nil {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	imgBytes, err := decodeImageBody(body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	ext := extFromContentType(r.Header.Get("Content-Type"))

	if err := api.playlists.SetImage(ctx, id, bytes.NewReader(imgBytes), ext); err != nil {
		api.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteItemImage removes a playlist's uploaded cover. Like postItemImage, only playlists
// are supported.
func (api *Router) deleteItemImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "itemId"))

	if _, err := api.playlists.Get(ctx, id); err != nil {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	if err := api.playlists.RemoveImage(ctx, id); err != nil {
		api.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// decodeImageBody returns the raw image bytes. Jellyfin sends the image base64-encoded in the
// request body; a few clients send raw bytes instead, so bytes already starting with a known
// image magic number are passed through as-is.
func decodeImageBody(body []byte) ([]byte, error) {
	if isImageMagic(body) {
		return body, nil
	}
	trimmed := bytes.TrimSpace(body)
	return base64.StdEncoding.DecodeString(string(trimmed))
}

func isImageMagic(b []byte) bool {
	switch {
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xD8: // JPEG
		return true
	case bytes.HasPrefix(b, []byte{0x89, 'P', 'N', 'G'}): // PNG
		return true
	default:
		return false
	}
}

// extFromContentType maps the client-supplied Content-Type to a file extension for storage;
// defaults to .jpg (Jellyfin's own default cover format) for anything unrecognized.
func extFromContentType(contentType string) string {
	ct, _, _ := strings.Cut(contentType, ";")
	switch strings.ToLower(strings.TrimSpace(ct)) {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		return ".jpg"
	}
}
