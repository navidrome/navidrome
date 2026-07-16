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
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

// playlistsFolderID is the reserved id of the synthetic "playlists library" folder. Clients resolve
// it via a ManualPlaylistsFolder query, then list playlists with ParentId set to it. The literal
// can't collide with real ids (those are hashes).
const playlistsFolderID = "playlists"

// playlistsFolder is the item returned for a ManualPlaylistsFolder query. CollectionType must be
// "playlists" — how the client identifies it; without it Jellify's playlist-library query loops.
func playlistsFolder() dto.BaseItemDto {
	return dto.BaseItemDto{
		Id:             dto.EncodeID(playlistsFolderID),
		Name:           "Playlists",
		Type:           "ManualPlaylistsFolder",
		CollectionType: "playlists",
		IsFolder:       true,
	}
}

// playlistError maps core/playlists write errors to HTTP status: ownership -> 403, missing/invisible
// -> 404 (never revealing another user's private playlist), else -> 500.
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

// createPlaylist always creates a new playlist (playlistId "" tells core/playlists.Create not to
// replace an existing one), owned by the authenticated user.
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

// updatePlaylistRequest mirrors Jellyfin's NewPlaylist body. Pointers so an absent field means
// "leave unchanged", distinguishing an omitted Ids (no change) from an explicit empty list (clear).
type updatePlaylistRequest struct {
	Name     *string   `json:"Name"`
	Ids      *[]string `json:"Ids"`
	IsPublic *bool     `json:"IsPublic"`
}

func (api *Router) updatePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	var body updatePlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// A present Ids replaces the track list. An empty list must clear it explicitly, since Create
	// can't persist an empty track list (the repository skips track writes when the list is empty).
	if body.Ids != nil {
		if len(*body.Ids) == 0 {
			if err := api.clearPlaylist(ctx, id); err != nil {
				api.playlistError(w, r, err)
				return
			}
		} else {
			ids := api.expandContainerIDs(ctx, slice.Map(*body.Ids, dto.DecodeID))
			if _, err := api.playlists.Create(ctx, id, "", ids); err != nil {
				api.playlistError(w, r, err)
				return
			}
		}
	}
	if body.Ids == nil || body.Name != nil || body.IsPublic != nil {
		if err := api.playlists.Update(ctx, id, body.Name, nil, body.IsPublic, nil, nil); err != nil {
			api.playlistError(w, r, err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

// clearPlaylist removes every track from a playlist. RemoveTracks enforces ownership.
func (api *Router) clearPlaylist(ctx context.Context, id string) error {
	pls, err := api.playlists.GetWithTracks(ctx, id)
	if err != nil {
		return err
	}
	if len(pls.Tracks) == 0 {
		return nil
	}
	entryIDs := slice.Map(pls.Tracks, func(t model.PlaylistTrack) string { return t.ID })
	return api.playlists.RemoveTracks(ctx, id, entryIDs)
}

// playlistTrackPage streams one page of a playlist's tracks. Streams because a playlist can be the
// whole library (a smart playlist matching everything) and clients may omit Limit. Excludes missing
// tracks, and counts the same set, like GetWithTracks.
func (api *Router) playlistTrackPage(repo model.PlaylistTrackRepository, fields dto.Fields, offset, limit int) (itemsResult, error) {
	total, err := repo.CountAll(model.QueryOptions{Filters: notMissing})
	if err != nil {
		return itemsResult{}, err
	}
	opts := model.QueryOptions{Sort: "id", Offset: offset, Max: limit, Filters: notMissing}
	open := streamCursor(func() (func(func(model.PlaylistTrack, error) bool), error) {
		return repo.GetCursor(opts)
	}, func(t model.PlaylistTrack) dto.BaseItemDto { return trackToBaseItem(t, fields) })
	return streamed(open, int(total), offset), nil
}

// trackToBaseItem maps a playlist entry to a BaseItemDto, tagging it with PlaylistItemId (the
// entry's id, model.PlaylistTrack.ID, not the song id). Clients echo it back via
// DELETE .../Items?EntryIds= to remove a specific occurrence, so duplicates of the same song remain
// individually removable.
func trackToBaseItem(t model.PlaylistTrack, fields dto.Fields) dto.BaseItemDto {
	item := dto.SongToBaseItem(t.MediaFile, fields)
	item.PlaylistItemId = dto.EncodeID(t.ID)
	return item
}

// getPlaylist returns a playlist's visibility flag and item ids (Finamp reads OpenAccess before the
// edit screen). Get and Tracks enforce visibility; any error maps to 404 so private playlists can't
// be probed.
func (api *Router) getPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	pls, err := api.playlists.Get(ctx, id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	repo, err := api.playlists.Tracks(ctx, id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	// PlaylistInfo carries every track id, so this can't be paged — but it needs no track data.
	trackIDs, err := repo.GetMediaFileIDs(model.QueryOptions{Sort: "id", Filters: notMissing})
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	itemIds := slice.Map(trackIDs, dto.EncodeID)
	api.ok(w, r, dto.PlaylistInfo{
		OpenAccess: pls.Public,
		Shares:     []dto.PlaylistUserPermissions{},
		ItemIds:    itemIds,
	})
}

// getPlaylistItems relies on Tracks to enforce visibility; any error maps to a generic 404 so a
// playlist id can't probe for private playlists.
func (api *Router) getPlaylistItems(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	repo, err := api.playlists.Tracks(ctx, id)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	p := req.Params(r)
	fields := dto.ParseFields(p.StringOr("fields", ""))
	res, err := api.playlistTrackPage(repo, fields, p.IntOr("startindex", 0), p.IntOr("limit", 0))
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// queryIDs reads an id-list query param that clients spell two ways: comma-separated in a single
// param (Finamp: ids=X,Y) or as repeated params (Jellify's @jellyfin/sdk: ids=X&ids=Y). It returns
// the flattened, non-empty ids across both forms.
func queryIDs(r *http.Request, key string) []string {
	var ids []string
	for _, v := range r.URL.Query()[key] {
		for id := range strings.SplitSeq(v, ",") {
			if id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// expandContainerIDs expands the container ids (albums, artists, playlists) a client sends when
// building a playlist into their track ids, in order, since core/playlists only understands media
// file ids. Unknown ids pass through unchanged. Songs are classified with one batched query; only
// the rest pays per-id container probes.
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

// addToPlaylist appends items by id, expanding containers into tracks (see expandContainerIDs).
// AddTracks enforces ownership; any error maps to 404.
func (api *Router) addToPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	ids := api.expandContainerIDs(ctx, slice.Map(queryIDs(r, "ids"), dto.DecodeID))
	if _, err := api.playlists.AddTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// removeFromPlaylist removes entries by entryIds — playlist-entry ids (PlaylistItemId), not media
// file ids, since RemoveTracks deletes playlist_tracks rows by that id. RemoveTracks enforces
// ownership; any error maps to 404.
func (api *Router) removeFromPlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "playlistId"))
	ids := slice.Map(queryIDs(r, "entryids"), dto.DecodeID)
	if err := api.playlists.RemoveTracks(ctx, id, ids); err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// getPlaylistUsers and getPlaylistUser answer client probes (e.g. Finamp) made before allowing
// edits. Navidrome has no per-playlist ACL, so every user is reported CanEdit; ownership is still
// enforced by AddTracks/RemoveTracks.
func (api *Router) getPlaylistUsers(w http.ResponseWriter, r *http.Request) {
	u, _ := request.UserFrom(r.Context())
	api.ok(w, r, []dto.PlaylistUserPermissions{{UserId: dto.EncodeID(u.ID), CanEdit: true}})
}

func (api *Router) getPlaylistUser(w http.ResponseWriter, r *http.Request) {
	userId := chi.URLParam(r, "userId")
	api.ok(w, r, dto.PlaylistUserPermissions{UserId: userId, CanEdit: true})
}
