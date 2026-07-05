package jellyfin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/slice"
)

type createPlaylistRequest struct {
	Name      string   `json:"Name"`
	Ids       []string `json:"Ids"`
	MediaType string   `json:"MediaType"`
}

// createPlaylist always creates a brand-new playlist (playlistId "" tells core/playlists.Create
// not to replace an existing one), owned by the authenticated user.
func (api *Router) createPlaylist(w http.ResponseWriter, r *http.Request) {
	var body createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	id, err := api.playlists.Create(r.Context(), "", body.Name, body.Ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.ok(w, r, map[string]string{"Id": id})
}

// trackToBaseItem maps a playlist entry to a BaseItemDto, tagging it with PlaylistItemId — the
// entry's position within the playlist (model.PlaylistTrack.ID) — rather than only the underlying
// song id. Jellyfin clients read this list, then echo PlaylistItemId back via
// DELETE .../Items?EntryIds=... to remove a specific occurrence; that's also the id
// core/playlists.RemoveTracks expects (see removeFromPlaylist), so the round trip works even when
// the same song appears more than once in the playlist.
func trackToBaseItem(t model.PlaylistTrack) dto.BaseItemDto {
	item := dto.SongToBaseItem(t.MediaFile)
	item.PlaylistItemId = t.ID
	return item
}

// getPlaylistItems relies on core/playlists.GetWithTracks to enforce playlist visibility
// (public or owned by the current user); any error — not found or not visible — is reported as
// a generic 404 so a playlist id can't be used to probe for the existence of private playlists.
func (api *Router) getPlaylistItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "playlistId")
	pls, err := api.playlists.GetWithTracks(ctx, id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	items := slice.Map(pls.Tracks, trackToBaseItem)
	api.ok(w, r, dto.QueryResult{Items: items, TotalRecordCount: len(items)})
}

// splitIds parses a comma-separated query parameter into ids, treating an absent or empty
// parameter as no ids rather than a single empty-string id.
func splitIds(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// addToPlaylist appends songs by their own id (core/playlists.AddTracks treats ids as media file
// ids). Ownership/editability is enforced by AddTracks itself; any error maps to 404.
func (api *Router) addToPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "playlistId")
	ids := splitIds(r.URL.Query().Get("Ids"))
	if _, err := api.playlists.AddTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// removeFromPlaylist removes entries by EntryIds. Unlike addToPlaylist's Ids, these must be
// playlist-entry ids (model.PlaylistTrack.ID / trackToBaseItem's PlaylistItemId) — the position
// of the entry within the playlist — because core/playlists.RemoveTracks deletes playlist_tracks
// rows by that id, not by media file id. Ownership/editability is enforced by RemoveTracks itself;
// any error maps to 404.
func (api *Router) removeFromPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "playlistId")
	ids := splitIds(r.URL.Query().Get("EntryIds"))
	if err := api.playlists.RemoveTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
