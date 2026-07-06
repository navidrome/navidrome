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
	// normalizeQueryKeys has folded every query key to lowercase, so params are read by their
	// lowercase name here (matching real Jellyfin's case-insensitive binding).
	//
	// Jellyfin's /Items?ids= is a batch-fetch-by-id that cuts across the type dispatch below
	// entirely; Finamp's download/sync uses it to fetch a single track's BaseItemDto.
	if ids := p.StringOr("ids", ""); ids != "" {
		var items []dto.BaseItemDto
		for rawID := range strings.SplitSeq(ids, ",") {
			if item, ok := api.resolveItemByID(ctx, dto.DecodeID(strings.TrimSpace(rawID))); ok {
				items = append(items, item)
			}
		}
		return result(items, len(items), 0), nil
	}
	parentId := dto.DecodeID(p.StringOr("parentid", ""))
	search := p.StringOr("searchterm", "")
	// Clients express "favorites only" two ways: Jellyfin's Filters=IsFavorite, and the standalone
	// isFavorite=true query param (Finamp's artist "Favourite tracks" widget uses the latter).
	favOnly := strings.Contains(p.StringOr("filters", ""), "IsFavorite") || p.BoolOr("isfavorite", false)
	sortBy := p.StringOr("sortby", "")
	sortOrder := p.StringOr("sortorder", "")
	offset := p.IntOr("startindex", 0)
	limit := p.IntOr("limit", 0)
	rawTypes := p.StringOr("includeitemtypes", "")
	// A ManualPlaylistsFolder query asks for the "playlists library" container, not real items.
	// Answer it with the synthetic folder so the client can then browse into it (see below).
	if strings.Contains(rawTypes, "ManualPlaylistsFolder") {
		return result([]dto.BaseItemDto{playlistsFolder()}, 1, 0), nil
	}
	types := parseTypes(rawTypes)
	// An artist's page filters by artist, not by ParentId: Finamp sends ParentId=<libraryId> for
	// scoping plus AlbumArtistIds/ArtistIds/contributingArtistIds for the artist itself. Without
	// this, an artist's albums/tracks come back unfiltered (all artists).
	//
	// albumArtistIds/artistIds select the artist's own discography; contributingArtistIds alone
	// selects albums the artist merely appears on (Jellyfin's "Featured On"), which must exclude
	// that discography — otherwise their own albums show up in both sections.
	albumArtistScope := firstNonEmpty(p.StringOr("albumartistids", ""), p.StringOr("artistids", ""))
	contributingScope := p.StringOr("contributingartistids", "")
	artistId := firstDecodedID(firstNonEmpty(albumArtistScope, contributingScope))
	contributingOnly := albumArtistScope == "" && contributingScope != ""

	scopeIDs, isLibraryParent := resolveLibraryScope(ctx, parentId)
	// When no item type is requested and ParentId is an album, browse into that album's tracks:
	// Jellify opens an album with just parentId=<albumId> (no IncludeItemTypes), and Jellyfin infers
	// the child type from the parent. Without this we'd fall back to parseTypes' MusicAlbum default
	// and list every album instead. An artist parent keeps that default (browse its albums).
	if rawTypes == "" && parentId != "" && !isLibraryParent {
		switch {
		case parentId == playlistsFolderID:
			// Browsing into the synthetic playlists folder lists the user's playlists.
			types = []string{"Playlist"}
		default:
			if _, err := api.ds.Album(ctx).Get(parentId); err == nil {
				types = []string{"Audio"}
			}
		}
	}
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
		return api.queryItemsOfType(ctx, types[0], opts, entityParent, artistId, contributingOnly, scopeIDs, search, favOnly)
	}

	var items []dto.BaseItemDto
	total := 0
	for _, itemType := range types {
		var opts model.QueryOptions
		applySort(&opts, itemType, sortBy, sortOrder)
		res, err := api.queryItemsOfType(ctx, itemType, opts, entityParent, artistId, contributingOnly, scopeIDs, search, favOnly)
		if err != nil {
			return dto.QueryResult{}, err
		}
		items = append(items, res.Items...)
		total += res.TotalRecordCount
	}
	return result(paginate(items, offset, limit), total, offset), nil
}

func (api *Router) queryItemsOfType(ctx context.Context, itemType string, opts model.QueryOptions, entityParent, artistId string, contributingOnly bool, scopeIDs []int, search string, favOnly bool) (dto.QueryResult, error) {
	switch itemType {
	case "Audio":
		return api.listSongs(ctx, opts, entityParent, artistId, scopeIDs, search, favOnly)
	case "MusicArtist":
		// The MusicArtist browse hierarchy (UserViews -> artists -> albums) means album artists.
		return api.listArtists(ctx, opts, scopeIDs, search, favOnly, model.RoleAlbumArtist)
	case "MusicGenre":
		return api.listGenres(ctx, opts)
	case "Playlist":
		return api.listPlaylists(ctx, opts, favOnly)
	default: // MusicAlbum
		return api.listAlbums(ctx, opts, entityParent, artistId, contributingOnly, scopeIDs, search, favOnly)
	}
}

// firstNonEmpty returns the first non-empty string, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// firstDecodedID decodes the first id from a (possibly comma-separated) Jellyfin id list.
func firstDecodedID(s string) string {
	if s == "" {
		return ""
	}
	first, _, _ := strings.Cut(s, ",")
	return dto.DecodeID(strings.TrimSpace(first))
}

// parseTypes returns every recognized entry in the (possibly comma-separated) IncludeItemTypes,
// in the order they appear, defaulting to []string{"MusicAlbum"} when nothing is recognized.
// The MusicAlbum default makes ParentId=<artistId> (no explicit type) browse into that artist's
// albums, matching the UserViews -> artists -> albums -> songs hierarchy. Browsing into an album
// with no explicit type (as Jellify does) is handled by queryItems, which infers Audio when
// ParentId names an album.
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

// searchPage runs a repository Search with one extra row beyond the requested page and derives
// TotalRecordCount from what came back: the repos' Search API returns a page without a match
// count, and CountAll can't see the search term. offset+len(rows) is exact once the matches end
// and a strictly growing lower bound before that, so a client that pages until StartIndex
// reaches TotalRecordCount terminates exactly at the last match.
func searchPage[S ~[]E, E any](opts model.QueryOptions, search func(model.QueryOptions) (S, error)) (S, int, error) {
	fetch := opts
	if fetch.Max > 0 {
		fetch.Max++
	}
	rows, err := search(fetch)
	if err != nil {
		return nil, 0, err
	}
	total := opts.Offset + len(rows)
	if opts.Max > 0 && len(rows) > opts.Max {
		rows = rows[:opts.Max]
	}
	return rows, total, nil
}

func (api *Router) listAlbums(ctx context.Context, opts model.QueryOptions, parentId, artistId string, contributingOnly bool, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Album(ctx)
	filters := squirrel.And{}
	// For albums, both ParentId (browse into an artist) and AlbumArtistIds/ArtistIds mean "this
	// artist's albums"; contributingArtistIds instead means "albums this artist only appears on"
	// (Featured On), which excludes their own discography.
	switch {
	case contributingOnly && artistId != "":
		filters = append(filters, filter.AlbumsByContributingArtistID(artistId).Filters)
	case firstNonEmpty(artistId, parentId) != "":
		filters = append(filters, filter.AlbumsByArtistID(firstNonEmpty(artistId, parentId)).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, scopeIDs)

	if search != "" {
		albums, total, err := searchPage(opts, func(o model.QueryOptions) (model.Albums, error) {
			return repo.Search(search, o)
		})
		if err != nil {
			return dto.QueryResult{}, err
		}
		return result(slice.Map(albums, dto.AlbumToBaseItem), total, opts.Offset), nil
	}
	albums, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(albums, dto.AlbumToBaseItem), int(total), opts.Offset), nil
}

func (api *Router) listSongs(ctx context.Context, opts model.QueryOptions, parentId, artistId string, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.MediaFile(ctx)
	filters := squirrel.And{}
	// For songs, ArtistIds/AlbumArtistIds selects an artist's tracks, while ParentId selects an
	// album's tracks — different filters.
	switch {
	case artistId != "":
		filters = append(filters, filter.SongsByArtistID(artistId).Filters)
	case parentId != "":
		filters = append(filters, filter.SongsByAlbum(parentId).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, scopeIDs)

	if search != "" {
		mfs, total, err := searchPage(opts, func(o model.QueryOptions) (model.MediaFiles, error) {
			return repo.Search(search, o)
		})
		if err != nil {
			return dto.QueryResult{}, err
		}
		return result(slice.Map(mfs, dto.SongToBaseItem), total, opts.Offset), nil
	}
	// When browsing an album's tracks, default to track order (disc + track number) like
	// Subsonic's GetAlbum and real Jellyfin do; an explicit SortBy from the client still wins,
	// since applySort would already have set opts.Sort.
	if artistId == "" && parentId != "" && opts.Sort == "" {
		opts.Sort = filter.SongsByAlbum(parentId).Sort
	}
	mfs, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(mfs, dto.SongToBaseItem), int(total), opts.Offset), nil
}

// listArtists lists artists in the given role — RoleAlbumArtist for the "album artists" views
// (/Artists/AlbumArtists, the MusicArtist browse hierarchy) and RoleArtist for performing artists
// (/Artists). Without a role filter every participant (composers, arrangers, ...) would show up in
// both lists and they'd be identical.
func (api *Router) listArtists(ctx context.Context, opts model.QueryOptions, scopeIDs []int, search string, fav bool, role model.Role) (dto.QueryResult, error) {
	repo := api.ds.Artist(ctx)

	// Artist Search does its own library scoping: it consumes a sole Eq{"library_id": ...} filter
	// (artists have no library_id column, so it can't be a real WHERE clause) and realizes it as a
	// search scope. The join-based ApplyArtistLibraryFilter, or any compound filter, would leak
	// library_artist.library_id into the FTS query and 500. So the search and browse paths build
	// their filters differently. See persistence/artist_repository.go's searchScope. Role isn't
	// applied to search: it's a name lookup, and a compound filter would break the same way.
	if search != "" {
		if len(scopeIDs) > 0 {
			opts.Filters = squirrel.Eq{"library_id": scopeIDs}
		}
		artists, total, err := searchPage(opts, func(o model.QueryOptions) (model.Artists, error) {
			return repo.Search(search, o)
		})
		if err != nil {
			return dto.QueryResult{}, err
		}
		return result(slice.Map(artists, dto.ArtistToBaseItem), total, opts.Offset), nil
	}

	if fav {
		opts.Filters = filter.ArtistsByStarred().Filters
	} else {
		opts.Filters = notMissing
	}
	opts = filter.ArtistsByRole(opts, role)
	opts = filter.ApplyArtistLibraryFilter(opts, scopeIDs)
	artists, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(artists, dto.ArtistToBaseItem), int(total), opts.Offset), nil
}

// listGenres is intentionally unscoped: genres are global tags derived from track metadata,
// not entities that belong to a single library. Paging happens in memory: GenreRepository has no
// CountAll, genre lists are small, and this way TotalRecordCount is the real total, not the page.
func (api *Router) listGenres(ctx context.Context, opts model.QueryOptions) (dto.QueryResult, error) {
	genres, err := api.ds.Genre(ctx).GetAll(model.QueryOptions{Sort: opts.Sort, Order: opts.Order})
	if err != nil {
		return dto.QueryResult{}, err
	}
	items := slice.Map(genres, dto.GenreToBaseItem)
	return result(paginate(items, opts.Offset, opts.Max), len(items), opts.Offset), nil
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

// resolveItemByID resolves an already-decoded navidrome id to its BaseItemDto, trying library
// view, album, artist, song and playlist in turn. For albums and songs (which each belong to
// exactly one library) it reports not-found if the current user lacks access to that library, so
// an id can't be used to probe content outside the user's libraries. Shared by getItem (single
// fetch) and queryItems' Ids batch-fetch.
func (api *Router) resolveItemByID(ctx context.Context, id string) (dto.BaseItemDto, bool) {
	u, _ := request.UserFrom(ctx)
	// Finamp resolves a /UserViews entry (Id=library id) by fetching it as a plain item; without
	// this, the home screen and every library tab 404 trying to probe it as an album/artist/song.
	if libID, err := strconv.Atoi(id); err == nil && u.HasLibraryAccess(libID) {
		for _, lib := range u.Libraries {
			if lib.ID == libID {
				return libraryView(lib), true
			}
		}
		// Admin bypass: Libraries is empty but access is granted to every library, so fetch the
		// real one instead of returning a placeholder.
		if lib, err := api.ds.Library(ctx).Get(libID); err == nil {
			return libraryView(*lib), true
		}
	}
	if al, err := api.ds.Album(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(al.LibraryID) {
			return dto.BaseItemDto{}, false
		}
		return dto.AlbumToBaseItem(*al), true
	}
	if ar, err := api.ds.Artist(ctx).Get(id); err == nil {
		// TODO: an artist can have content in multiple libraries (via library_artist), so
		// there's no single LibraryID to check here; access control for artists relies on
		// list-time scoping (listArtists) and the persistence layer's defense-in-depth.
		return dto.ArtistToBaseItem(*ar), true
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(mf.LibraryID) {
			return dto.BaseItemDto{}, false
		}
		return dto.SongToBaseItem(*mf), true
	}
	// api.playlists.Get enforces ownership/visibility itself, so a non-owned or missing playlist
	// id falls through to the generic not-found below, same as every other probe here.
	if pl, err := api.playlists.Get(ctx, id); err == nil {
		return dto.PlaylistToBaseItem(*pl), true
	}
	return dto.BaseItemDto{}, false
}

func (api *Router) getItem(w http.ResponseWriter, r *http.Request) {
	id := dto.DecodeID(chi.URLParam(r, "itemId"))
	if item, ok := api.resolveItemByID(r.Context(), id); ok {
		api.ok(w, r, item)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

// deleteItem handles DELETE /Items/{id}. Only playlists are deletable through this API — albums and
// songs come from library scanning, not the client — so any non-playlist id resolves to 404.
// core/playlists.Delete enforces ownership (checkWritable) and removes the cover file.
func (api *Router) deleteItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := dto.DecodeID(chi.URLParam(r, "itemId"))
	if err := api.playlists.Delete(ctx, id); err != nil {
		api.playlistError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (api *Router) getLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	opts := filter.AlbumsByNewest()
	opts.Max = req.Params(r).IntOr("limit", 20)
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
		"album": "album",
		// Finamp's album view sorts by ParentIndexNumber,IndexNumber (disc, track). Navidrome's
		// "album" sort key is order_album_name, album_id, disc_number, track_number, ..., which is
		// exactly disc+track order within an album, so map both here.
		"indexnumber":       "album",
		"parentindexnumber": "album",
		"artist":            "artist",
		"albumartist":       "album_artist",
		"datecreated":       "recently_added",
		"playcount":         "play_count",
		"dateplayed":        "play_date",
		"communityrating":   "rating",
		"random":            "random",
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
	"Playlist": {
		"sortname": "name", "name": "name",
		"datecreated": "created_at",
	},
}

// sortColumn maps a single (non comma-list) Jellyfin SortBy key to the repo sort key for
// itemType, reporting false when it isn't recognized for that type.
func sortColumn(itemType, sortBy string) (string, bool) {
	col, ok := sortColumnsByType[itemType][strings.ToLower(sortBy)]
	return col, ok
}
