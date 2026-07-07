package jellyfin

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strconv"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	_ "golang.org/x/image/webp"
)

func (api *Router) getItemImage(w http.ResponseWriter, r *http.Request) {
	// Public endpoint (no user in ctx): library artwork isn't user-sensitive, so resolution runs
	// under an elevated context to bypass the persistence visibility filter; playlist access is
	// gated inside resolveArtworkID.
	ctx := request.WithUser(r.Context(), model.User{IsAdmin: true})
	itemId := api.resolveItemID(ctx, dto.DecodeID(chi.URLParam(r, "itemId")))
	size, _ := strconv.Atoi(r.URL.Query().Get("maxwidth"))

	artID := api.resolveArtworkID(ctx, r, itemId)
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
	// Leave Content-Type unset so net/http sniffs it (covers may be PNG/WebP/JPEG).
	_, _ = io.Copy(w, reader)
}

// resolveArtworkID maps a Jellyfin item id to a Navidrome ArtworkID, probing
// album -> artist -> media file -> playlist.
func (api *Router) resolveArtworkID(ctx context.Context, r *http.Request, itemId string) string {
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
		// Playlist covers are user-scoped: serve a private one only for a public playlist or a
		// token identifying its owner/an admin, so this public route can't probe others' covers.
		u, ok := api.userFromToken(r)
		if pl.Public || (ok && (u.IsAdmin || pl.OwnerID == u.ID)) {
			return pl.CoverArtID().String()
		}
	}
	return (model.ArtworkID{}).String()
}

// postItemImage handles cover upload. Only playlists are writable here; album/artist covers come
// from scanning. The body is always drained first (even on the not-implemented path) because
// Finamp writes it synchronously and sees a broken pipe if we respond before reading it.
func (api *Router) postItemImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "itemId"))

	// Honor the same artwork-upload gate and size cap as the native endpoint.
	u, _ := request.UserFrom(ctx)
	if !conf.Server.EnableArtworkUpload && !u.IsAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	// The limit caps the decoded image (native endpoint semantics); Jellyfin clients base64-encode
	// the wire body (4/3 bigger), so the read cap allows for inflation.
	limit := core.MaxImageUploadSize()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit*4/3+4))
	if err != nil {
		log.Warn(ctx, "Jellyfin API: cover upload rejected: body exceeds MaxImageUploadSize",
			"playlistId", id, "limit", humanize.Bytes(uint64(limit)), err)
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}

	if _, err := api.playlists.Get(ctx, id); err != nil {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	imgBytes, err := decodeImageBody(body)
	if err != nil {
		log.Warn(ctx, "Jellyfin API: cover upload rejected: body is neither an image nor base64", "playlistId", id, err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if int64(len(imgBytes)) > limit {
		log.Warn(ctx, "Jellyfin API: cover upload rejected: image exceeds MaxImageUploadSize",
			"playlistId", id, "size", humanize.Bytes(uint64(len(imgBytes))), "limit", humanize.Bytes(uint64(limit)))
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	// Validate by decoding and derive the extension from the real format — clients lie in Content-Type.
	_, format, err := image.DecodeConfig(bytes.NewReader(imgBytes))
	if err != nil {
		log.Warn(ctx, "Jellyfin API: cover upload rejected: not a valid image", "playlistId", id, err)
		http.Error(w, "invalid image file", http.StatusBadRequest)
		return
	}
	ext := "." + format

	if err := api.playlists.SetImage(ctx, id, bytes.NewReader(imgBytes), ext); err != nil {
		api.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteItemImage removes a playlist's uploaded cover. Only playlists are supported.
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

// decodeImageBody returns the raw image bytes. Jellyfin base64-encodes the body, but some clients
// send raw bytes, so input already starting with an image magic number is passed through as-is.
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
	case bytes.HasPrefix(b, []byte("GIF8")): // GIF (GIF87a/GIF89a)
		return true
	case len(b) >= 12 && bytes.HasPrefix(b, []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")): // WebP
		return true
	default:
		return false
	}
}
