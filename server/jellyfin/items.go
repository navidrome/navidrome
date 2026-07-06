package jellyfin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/filter"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

// notMissing excludes items whose backing files are all gone. It's hand-rolled rather than
// pulled from a filter.XxxByName()-style builder because none of those return just this
// condition; "missing" is a real column on album, artist and media_file (see persistence/).
var notMissing = squirrel.Eq{"missing": false}

func (api *Router) getItems(w http.ResponseWriter, r *http.Request) {
	res, err := api.queryItems(r.Context(), r)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// queryItems is the universal /Items dispatcher. It parses every recognized entity type out of
// IncludeItemTypes (falling back to MusicAlbum when none are recognized), queries each type via
// the matching listXxx, and — when more than one type was requested — merges the results into a
// single paginated list, as real Finamp does for its favorites screen
// (IncludeItemTypes=Audio,MusicAlbum,Playlist&Filters=IsFavorite).
func (api *Router) queryItems(ctx context.Context, r *http.Request) (dto.QueryResult, error) {
	p := req.Params(r)
	parentId := p.StringOr("ParentId", "")
	search := p.StringOr("SearchTerm", "")
	favOnly := strings.Contains(p.StringOr("Filters", ""), "IsFavorite")
	sortBy := p.StringOr("SortBy", "")
	sortOrder := p.StringOr("SortOrder", "")
	offset := p.IntOr("StartIndex", 0)
	limit := p.IntOr("Limit", 0)
	types := parseTypes(p.StringOr("IncludeItemTypes", ""))

	scopeIDs, isLibraryParent := resolveLibraryScope(ctx, parentId)
	entityParent := parentId
	// ParentId-as-entity-id (an artist for a MusicAlbum query, an album for an Audio query) only
	// makes sense when browsing a single type; a multi-type query has no single natural parent
	// entity type, so there ParentId only ever acts as library scoping.
	if isLibraryParent || len(types) > 1 {
		entityParent = ""
	}

	if len(types) == 1 {
		opts := model.QueryOptions{Offset: offset, Max: limit}
		applySort(&opts, types[0], sortBy, sortOrder)
		return api.queryItemsOfType(ctx, types[0], opts, entityParent, scopeIDs, search, favOnly)
	}

	var items []dto.BaseItemDto
	total := 0
	for _, itemType := range types {
		var opts model.QueryOptions
		applySort(&opts, itemType, sortBy, sortOrder)
		res, err := api.queryItemsOfType(ctx, itemType, opts, entityParent, scopeIDs, search, favOnly)
		if err != nil {
			return dto.QueryResult{}, err
		}
		items = append(items, res.Items...)
		total += res.TotalRecordCount
	}
	return result(paginate(items, offset, limit), total, offset), nil
}

func (api *Router) queryItemsOfType(ctx context.Context, itemType string, opts model.QueryOptions, entityParent string, scopeIDs []int, search string, favOnly bool) (dto.QueryResult, error) {
	switch itemType {
	case "Audio":
		return api.listSongs(ctx, opts, entityParent, scopeIDs, search, favOnly)
	case "MusicArtist":
		return api.listArtists(ctx, opts, scopeIDs, search, favOnly)
	case "MusicGenre":
		return api.listGenres(ctx, opts)
	case "Playlist":
		return api.listPlaylists(ctx, opts, favOnly)
	default: // MusicAlbum
		return api.listAlbums(ctx, opts, entityParent, scopeIDs, search, favOnly)
	}
}

// parseTypes returns every recognized entry in the (possibly comma-separated) IncludeItemTypes,
// in the order they appear, defaulting to []string{"MusicAlbum"} when nothing is recognized.
// The MusicAlbum default makes ParentId=<artistId> (no explicit type) browse into that artist's
// albums, matching the UserViews -> artists -> albums -> songs hierarchy; callers browsing into
// an album are expected to pass IncludeItemTypes=Audio explicitly, as Finamp and Jellyfin's own
// clients do.
func parseTypes(types string) []string {
	var recognized []string
	for t := range strings.SplitSeq(types, ",") {
		t = strings.TrimSpace(t)
		switch t {
		case "Audio", "MusicArtist", "MusicAlbum", "MusicGenre", "Playlist":
			recognized = append(recognized, t)
		}
	}
	if len(recognized) == 0 {
		return []string{"MusicAlbum"}
	}
	return recognized
}

// paginate applies StartIndex/Limit to an already-fetched, in-memory item list. It's only used
// for the multi-type merge path; single-type queries push Offset/Max down to the SQL query
// instead, which is why this isn't just folded into queryItems.
func paginate(items []dto.BaseItemDto, offset, limit int) []dto.BaseItemDto {
	if offset >= len(items) {
		return []dto.BaseItemDto{}
	}
	items = items[offset:]
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	return items
}

func (api *Router) listAlbums(ctx context.Context, opts model.QueryOptions, parentId string, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Album(ctx)
	filters := squirrel.And{}
	if parentId != "" {
		filters = append(filters, filter.AlbumsByArtistID(parentId).Filters)
	} else {
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, scopeIDs)

	var albums model.Albums
	var err error
	if search != "" {
		albums, err = repo.Search(search, opts)
	} else {
		albums, err = repo.GetAll(opts)
	}
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(albums, dto.AlbumToBaseItem), int(total), opts.Offset), nil
}

func (api *Router) listSongs(ctx context.Context, opts model.QueryOptions, parentId string, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.MediaFile(ctx)
	filters := squirrel.And{}
	if parentId != "" {
		filters = append(filters, filter.SongsByAlbum(parentId).Filters)
	} else {
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, scopeIDs)

	var mfs model.MediaFiles
	var err error
	if search != "" {
		mfs, err = repo.Search(search, opts)
	} else {
		mfs, err = repo.GetAll(opts)
	}
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(mfs, dto.SongToBaseItem), int(total), opts.Offset), nil
}

func (api *Router) listArtists(ctx context.Context, opts model.QueryOptions, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Artist(ctx)
	if fav {
		opts.Filters = filter.ArtistsByStarred().Filters
	} else {
		opts.Filters = notMissing
	}
	opts = filter.ApplyArtistLibraryFilter(opts, scopeIDs)

	var artists model.Artists
	var err error
	if search != "" {
		artists, err = repo.Search(search, opts)
	} else {
		artists, err = repo.GetAll(opts)
	}
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(artists, dto.ArtistToBaseItem), int(total), opts.Offset), nil
}

// listGenres is intentionally unscoped: genres are global tags derived from track metadata,
// not entities that belong to a single library.
func (api *Router) listGenres(ctx context.Context, opts model.QueryOptions) (dto.QueryResult, error) {
	genres, err := api.ds.Genre(ctx).GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	return result(slice.Map(genres, dto.GenreToBaseItem), len(genres), opts.Offset), nil
}

// listPlaylists lists playlists visible to the current user. Unlike albums/songs/artists,
// playlists aren't scoped by library — visibility (public, or owned by the current user) is
// enforced by persistence's playlistRepository itself, not by scopeIDs here. model.Playlist also
// carries no annotations (no starred concept), so a favorites query can never match a playlist;
// rather than send Filters=IsFavorite to a repo that doesn't understand it, short-circuit to an
// empty result.
func (api *Router) listPlaylists(ctx context.Context, opts model.QueryOptions, favOnly bool) (dto.QueryResult, error) {
	if favOnly {
		return result(nil, 0, opts.Offset), nil
	}
	repo := api.ds.Playlist(ctx)
	playlists, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(playlists, dto.PlaylistToBaseItem), int(total), opts.Offset), nil
}

// getItem fetches a single entity by id, trying library view, album, artist and song in turn.
// For albums and songs (which each belong to exactly one library) it 404s if the current user
// lacks access to that library, so an id can't be used to probe content outside the user's
// libraries.
func (api *Router) getItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "itemId")
	u, _ := request.UserFrom(ctx)
	// Finamp resolves a /UserViews entry (Id=library id) by fetching it as a plain item; without
	// this, the home screen and every library tab 404 trying to probe it as an album/artist/song.
	if libID, err := strconv.Atoi(id); err == nil && u.HasLibraryAccess(libID) {
		for _, lib := range u.Libraries {
			if lib.ID == libID {
				api.ok(w, r, libraryView(lib))
				return
			}
		}
		// Admin bypass: Libraries is empty but access is granted to every library, so fetch the
		// real one instead of returning a placeholder.
		if lib, err := api.ds.Library(ctx).Get(libID); err == nil {
			api.ok(w, r, libraryView(*lib))
			return
		}
	}
	if al, err := api.ds.Album(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(al.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		api.ok(w, r, dto.AlbumToBaseItem(*al))
		return
	}
	if ar, err := api.ds.Artist(ctx).Get(id); err == nil {
		// TODO: an artist can have content in multiple libraries (via library_artist), so
		// there's no single LibraryID to check here; access control for artists relies on
		// list-time scoping (listArtists) and the persistence layer's defense-in-depth.
		api.ok(w, r, dto.ArtistToBaseItem(*ar))
		return
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(mf.LibraryID) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		api.ok(w, r, dto.SongToBaseItem(*mf))
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

func (api *Router) getLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	opts := filter.AlbumsByNewest()
	opts.Max = req.Params(r).IntOr("Limit", 20)
	opts = filter.ApplyLibraryFilter(opts, accessibleLibraryIDs(ctx))
	albums, err := api.ds.Album(ctx).GetAll(opts)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, slice.Map(albums, dto.AlbumToBaseItem)) // /Latest returns a bare array
}

func result(items []dto.BaseItemDto, total, start int) dto.QueryResult {
	if items == nil {
		items = []dto.BaseItemDto{}
	}
	return dto.QueryResult{Items: items, TotalRecordCount: total, StartIndex: start}
}

// applySort translates Jellyfin's SortBy/SortOrder into a model.QueryOptions sort key valid for
// the given item type (see sortColumnsByType). Real clients (Finamp included) send SortBy as a
// comma-separated list of fallback keys, e.g. "DateCreated,SortName" or
// "ParentIndexNumber,IndexNumber,SortName" — this uses the first recognized key and ignores the
// rest. An unrecognized SortBy leaves opts.Sort untouched (the repo's own default), rather than
// passing it through raw and risking an invalid ORDER BY.
func applySort(opts *model.QueryOptions, itemType, sortBy, order string) {
	for key := range strings.SplitSeq(sortBy, ",") {
		if col, ok := sortColumn(itemType, strings.TrimSpace(key)); ok {
			opts.Sort = col
			break
		}
	}
	if strings.EqualFold(order, "Descending") {
		opts.Order = "desc"
	}
}

// sortColumnsByType is a lowercased-SortBy -> repo-sort-key table per item type. A map (rather
// than a switch per type) keeps this a flat data lookup instead of a large branch, since each
// repository maps different logical fields to different (or annotation-joined) real columns —
// e.g. media_file has no "name" column, only "title"; artist has no "random" column.
var sortColumnsByType = map[string]map[string]string{
	"Audio": {
		"sortname": "title", "name": "title",
		"album":           "album",
		"artist":          "artist",
		"albumartist":     "album_artist",
		"datecreated":     "recently_added",
		"playcount":       "play_count",
		"dateplayed":      "play_date",
		"communityrating": "rating",
		"random":          "random",
	},
	"MusicArtist": {
		"sortname": "name", "name": "name",
		"albumcount":      "album_count",
		"songcount":       "song_count",
		"datecreated":     "created_at",
		"playcount":       "play_count",
		"dateplayed":      "play_date",
		"communityrating": "rating",
	},
	"MusicAlbum": {
		"sortname": "name", "name": "name", "album": "name",
		"artist":          "artist",
		"albumartist":     "album_artist",
		"datecreated":     "recently_added",
		"random":          "random",
		"playcount":       "play_count",
		"dateplayed":      "play_date",
		"communityrating": "rating",
		"premieredate":    "max_year", "productionyear": "max_year",
	},
	"MusicGenre": {
		"sortname": "name", "name": "name",
	},
}

// sortColumn maps a single (non comma-list) Jellyfin SortBy key to the repo sort key for
// itemType, reporting false when it isn't recognized for that type.
func sortColumn(itemType, sortBy string) (string, bool) {
	col, ok := sortColumnsByType[itemType][strings.ToLower(sortBy)]
	return col, ok
}
