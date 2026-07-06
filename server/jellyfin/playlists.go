package jellyfin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/filter"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/slice"
)

// playlistsFolderID is the reserved id of the synthetic "playlists library" folder. Jellyfin
// clients (Jellify) first resolve this folder via an IncludeItemTypes=ManualPlaylistsFolder query,
// then list playlists with ParentId set to its id. Navidrome has no such container, so it's
// modelled here. The literal can't collide with a real album/artist/song id (those are hashes).
const playlistsFolderID = "playlists"

// playlistsFolder is the single item returned for a ManualPlaylistsFolder query. Its CollectionType
// must be "playlists" — that's how the client picks it out. Without it, the client's playlist-library
// query resolves to undefined and (because React Query rejects undefined results) retries in a
// backoff loop that stalls the home screen.
func playlistsFolder() dto.BaseItemDto {
	return dto.BaseItemDto{
		Id:             dto.EncodeID(playlistsFolderID),
		Name:           "Playlists",
		Type:           "ManualPlaylistsFolder",
		CollectionType: "playlists",
		IsFolder:       true,
	}
}

// playlistError maps core/playlists write errors to an HTTP status: ownership -> 403, missing or
// invisible -> 404 (never revealing another user's private playlist), anything else -> 500.
// Shared by the playlist mutation handlers.
func (api *Router) playlistError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, model.ErrNotAuthorized):
		http.Error(w, "Forbidden", http.StatusForbidden)
	case errors.Is(err, model.ErrNotFound):
		http.Error(w, "Not Found", http.StatusNotFound)
	default:
		api.internalError(w, r, err)
	}
}

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
	ids := api.expandContainerIDs(r.Context(), slice.Map(body.Ids, dto.DecodeID))
	id, err := api.playlists.Create(r.Context(), "", body.Name, ids)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, map[string]string{"Id": dto.EncodeID(id)})
}

// updatePlaylistRequest mirrors Jellyfin's NewPlaylist body reused for updates. Name/IsPublic are
// pointers so an absent field means "leave unchanged"; Finamp sends name/visibility edits and
// track-order edits as two separate requests, never combined.
type updatePlaylistRequest struct {
	Name     *string  `json:"Name"`
	Ids      []string `json:"Ids"`
	IsPublic *bool    `json:"IsPublic"`
}

// updatePlaylist handles POST /Playlists/{id}, applying every provided field (Jellyfin's
// UpdatePlaylist semantics): Ids replace the playlist's track list, Name/IsPublic update
// metadata, and a combined body applies both. Ownership is enforced by core/playlists.
func (api *Router) updatePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	var body updatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Create with an existing id replaces the tracks and keeps the name; expandContainerIDs lets a
	// client send containers here too, consistent with create/add.
	if len(body.Ids) > 0 {
		ids := api.expandContainerIDs(ctx, slice.Map(body.Ids, dto.DecodeID))
		if _, err := api.playlists.Create(ctx, id, "", ids); err != nil {
			api.playlistError(w, r, err)
			return
		}
	}
	if len(body.Ids) == 0 || body.Name != nil || body.IsPublic != nil {
		if err := api.playlists.Update(ctx, id, body.Name, nil, body.IsPublic, nil, nil); err != nil {
			api.playlistError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// trackToBaseItem maps a playlist entry to a BaseItemDto, tagging it with PlaylistItemId — the
// entry's position within the playlist (model.PlaylistTrack.ID) — rather than only the underlying
// song id. Jellyfin clients read this list, then echo PlaylistItemId back via
// DELETE .../Items?EntryIds=... to remove a specific occurrence; that's also the id
// core/playlists.RemoveTracks expects (see removeFromPlaylist), so the round trip works even when
// the same song appears more than once in the playlist.
func trackToBaseItem(t model.PlaylistTrack) dto.BaseItemDto {
	item := dto.SongToBaseItem(t.MediaFile)
	item.PlaylistItemId = dto.EncodeID(t.ID)
	return item
}

// getPlaylist returns a playlist's public-visibility flag and item ids. Finamp calls this before
// opening a playlist's edit screen to read OpenAccess. Visibility is enforced by GetWithTracks
// (public or owned by the current user); any error maps to 404 so private playlists can't be probed.
func (api *Router) getPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	pls, err := api.playlists.GetWithTracks(ctx, id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	itemIds := slice.Map(pls.Tracks, func(t model.PlaylistTrack) string { return dto.EncodeID(t.MediaFileID) })
	api.ok(w, r, dto.PlaylistInfo{
		OpenAccess: pls.Public,
		Shares:     []dto.PlaylistUserPermissions{},
		ItemIds:    itemIds,
	})
}

// getPlaylistItems relies on core/playlists.GetWithTracks to enforce playlist visibility
// (public or owned by the current user); any error — not found or not visible — is reported as
// a generic 404 so a playlist id can't be used to probe for the existence of private playlists.
func (api *Router) getPlaylistItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
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

// expandContainerIDs turns the ids a Jellyfin client sends when building a playlist into the
// underlying media file ids. Clients populate the id list with containers — albums, artists,
// playlists — not just songs, and expect the server to expand each into its tracks, in order.
// core/playlists only understands media file ids, so an unexpanded album id would silently add
// nothing. A bare song id (or any id matching no container) passes through unchanged.
// Songs (the common case, e.g. a 1000-track bulk add) are classified with one batched query;
// only the rest pays the per-id container probes.
func (api *Router) expandContainerIDs(ctx context.Context, ids []string) []string {
	songs := api.songsByIDs(ctx, ids)
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if _, ok := songs[id]; ok {
			out = append(out, id) // already a song
		} else if _, err := api.ds.Album(ctx).Get(id); err == nil {
			out = append(out, api.songIDs(ctx, filter.SongsByAlbum(id))...)
		} else if _, err := api.ds.Artist(ctx).Get(id); err == nil {
			out = append(out, api.songIDs(ctx, filter.SongsByArtistID(id))...)
		} else if pl, err := api.playlists.GetWithTracks(ctx, id); err == nil {
			out = append(out, slice.Map(pl.Tracks, func(t model.PlaylistTrack) string { return t.MediaFileID })...)
		} else {
			out = append(out, id) // unknown id — pass through unchanged
		}
	}
	return out
}

func (api *Router) songIDs(ctx context.Context, opts model.QueryOptions) []string {
	mfs, err := api.ds.MediaFile(ctx).GetAll(opts)
	if err != nil {
		log.Error(ctx, "Jellyfin: error expanding container to tracks", err)
		return nil
	}
	return slice.Map(mfs, func(mf model.MediaFile) string { return mf.ID })
}

// addToPlaylist appends items by id, expanding album/artist/playlist containers into their tracks
// (see expandContainerIDs). Ownership/editability is enforced by AddTracks itself; any error maps
// to 404.
func (api *Router) addToPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	ids := api.expandContainerIDs(ctx, slice.Map(splitIds(r.URL.Query().Get("ids")), dto.DecodeID))
	if _, err := api.playlists.AddTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// removeFromPlaylist removes entries by entryIds. Unlike addToPlaylist's ids, these must be
// playlist-entry ids (model.PlaylistTrack.ID / trackToBaseItem's PlaylistItemId) — the position
// of the entry within the playlist — because core/playlists.RemoveTracks deletes playlist_tracks
// rows by that id, not by media file id. Ownership/editability is enforced by RemoveTracks itself;
// any error maps to 404.
func (api *Router) removeFromPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	ids := slice.Map(splitIds(r.URL.Query().Get("entryids")), dto.DecodeID)
	if err := api.playlists.RemoveTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getPlaylistUsers and getPlaylistUser are best-effort probes some clients (e.g. Finamp) make
// before allowing playlist edits. Navidrome has no per-playlist ACL model, so every user is
// reported as able to edit; ownership is still enforced by AddTracks/RemoveTracks themselves.
func (api *Router) getPlaylistUsers(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	api.ok(w, r, []dto.PlaylistUserPermissions{{UserId: u.ID, CanEdit: true}})
}

func (api *Router) getPlaylistUser(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	api.ok(w, r, dto.PlaylistUserPermissions{UserId: userId, CanEdit: true})
}
