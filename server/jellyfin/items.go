package jellyfin

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
	"github.com/navidrome/navidrome/server/filter"
	"github.com/navidrome/navidrome/server/jellyfin/dto"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
)

// notMissing excludes items whose backing files are all gone ("missing" is a real column on
// album, artist and media_file).
var notMissing = squirrel.Eq{"missing": false}

func (api *Router) getItems(w http.ResponseWriter, r *http.Request) {
	res, err := api.queryItems(r.Context(), r)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// queryItems is the /Items dispatcher: it parses entity types from IncludeItemTypes (defaulting to
// MusicAlbum), queries each via the matching listXxx, and merges multi-type results into one
// paginated list (as Finamp's favorites screen requests).
func (api *Router) queryItems(ctx context.Context, r *http.Request) (dto.QueryResult, error) {
	p := req.Params(r)
	// Query keys are read lowercase because normalizeQueryKeys folded them (Jellyfin binds
	// case-insensitively). /Items?ids= is a batch-fetch-by-id that bypasses the type dispatch below.
	fields := dto.ParseFields(p.StringOr("fields", ""))
	if ids := decodedQueryIDs(r, "ids"); len(ids) > 0 {
		return api.itemsByIDs(ctx, ids, fields), nil
	}
	parentId := dto.DecodeID(p.StringOr("parentid", ""))
	search := p.StringOr("searchterm", "")
	// Clients express "favorites only" two ways: Filters=IsFavorite and the standalone
	// isFavorite=true param (Finamp's "Favourite tracks" widget uses the latter).
	favOnly := strings.Contains(p.StringOr("filters", ""), "IsFavorite") || p.BoolOr("isfavorite", false)
	sortBy := p.StringOr("sortby", "")
	sortOrder := p.StringOr("sortorder", "")
	offset := p.IntOr("startindex", 0)
	limit := p.IntOr("limit", 0)
	rawTypes := p.StringOr("includeitemtypes", "")
	// A ManualPlaylistsFolder query asks for the synthetic "playlists library" container, not real items.
	if strings.Contains(rawTypes, "ManualPlaylistsFolder") {
		return result([]dto.BaseItemDto{playlistsFolder()}, 1, 0), nil
	}
	types := parseTypes(rawTypes)
	// An artist's page filters by artist, not ParentId: Finamp sends ParentId=<libraryId> for scoping
	// plus AlbumArtistIds/ArtistIds/contributingArtistIds for the artist. albumArtistIds/artistIds
	// select the artist's own discography; contributingArtistIds alone means albums they merely appear
	// on (Jellyfin's "Featured On"), which must exclude that discography.
	albumArtistScope := firstNonEmpty(p.StringOr("albumartistids", ""), p.StringOr("artistids", ""))
	contributingScope := p.StringOr("contributingartistids", "")
	artistId := firstDecodedID(firstNonEmpty(albumArtistScope, contributingScope))
	contributingOnly := albumArtistScope == "" && contributingScope != ""
	// Finamp's genre screen sends ParentId=<libraryId> for scoping plus GenreIds for the genre.
	genreIds := decodedQueryIDs(r, "genreids")

	scopeIDs, isLibraryParent := resolveLibraryScope(ctx, parentId)
	// A playlist parent always resolves to its tracks, whatever IncludeItemTypes says. Jellify opens
	// a playlist with ParentId=<playlist>&IncludeItemTypes=Audio; routing that through listSongs would
	// treat the playlist id as an album id and return nothing.
	if parentId != "" && !isLibraryParent && parentId != playlistsFolderID {
		if pls, err := api.playlists.GetWithTracks(ctx, parentId); err == nil {
			// GetWithTracks enforces visibility (public or owned by the current user).
			items := slice.Map(pls.Tracks, func(t model.PlaylistTrack) dto.BaseItemDto { return trackToBaseItem(t, fields) })
			return result(paginate(items, offset, limit), len(items), offset), nil
		}
	}
	// With no item type, Jellyfin infers the child type from the parent: album parent -> its tracks
	// (Jellify opens albums this way). An artist parent keeps parseTypes' MusicAlbum default (browse
	// its albums).
	if rawTypes == "" && parentId != "" && !isLibraryParent {
		if parentId == playlistsFolderID {
			// Browsing into the synthetic playlists folder lists the user's playlists.
			types = []string{"Playlist"}
		} else if _, err := api.ds.Album(ctx).Get(parentId); err == nil {
			types = []string{"Audio"}
		}
	}
	entityParent := parentId
	// ParentId-as-entity-id (artist for MusicAlbum, album for Audio) only makes sense for a single
	// type; a multi-type query has no natural parent entity, so ParentId is only library scoping there.
	if isLibraryParent || len(types) > 1 {
		entityParent = ""
	}

	if len(types) == 1 {
		opts := model.QueryOptions{Offset: offset, Max: limit}
		applySort(&opts, types[0], sortBy, sortOrder)
		return api.queryItemsOfType(ctx, types[0], opts, entityParent, artistId, contributingOnly, genreIds, scopeIDs, search, favOnly, fields)
	}

	var items []dto.BaseItemDto
	total := 0
	for _, itemType := range types {
		var opts model.QueryOptions
		// Each per-type query needs at most offset+limit rows (the worst case where one type fills the
		// whole [offset, offset+limit) window); without this cap each would fetch its whole table.
		// Totals are unaffected — they come from CountAll.
		if limit > 0 {
			opts.Max = offset + limit
		}
		applySort(&opts, itemType, sortBy, sortOrder)
		res, err := api.queryItemsOfType(ctx, itemType, opts, entityParent, artistId, contributingOnly, genreIds, scopeIDs, search, favOnly, fields)
		if err != nil {
			return dto.QueryResult{}, err
		}
		items = append(items, res.Items...)
		total += res.TotalRecordCount
	}
	return result(paginate(items, offset, limit), total, offset), nil
}

func (api *Router) queryItemsOfType(ctx context.Context, itemType string, opts model.QueryOptions, entityParent, artistId string, contributingOnly bool, genreIds []string, scopeIDs []int, search string, favOnly bool, fields dto.Fields) (dto.QueryResult, error) {
	switch itemType {
	case "Audio":
		return api.listSongs(ctx, opts, entityParent, artistId, genreIds, scopeIDs, search, favOnly, fields)
	case "MusicArtist":
		// The MusicArtist browse hierarchy (UserViews -> artists -> albums) means album artists.
		return api.listArtists(ctx, opts, genreIds, scopeIDs, search, favOnly, model.RoleAlbumArtist)
	case "MusicGenre":
		return api.listGenres(ctx, opts)
	case "Playlist":
		return api.listPlaylists(ctx, opts, favOnly)
	default: // MusicAlbum
		return api.listAlbums(ctx, opts, entityParent, artistId, contributingOnly, genreIds, scopeIDs, search, favOnly)
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

// decodedQueryIDs reads an id-list param in both client spellings (see queryIDs), decoding each id.
func decodedQueryIDs(r *http.Request, key string) []string {
	return slice.Map(queryIDs(r, key), dto.DecodeID)
}

// parseTypes returns the recognized entries in IncludeItemTypes in order, defaulting to
// {"MusicAlbum"} when none are recognized (so ParentId=<artistId> browses that artist's albums).
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

// paginate applies StartIndex/Limit to an in-memory item list, for the multi-type merge path only
// (single-type queries push Offset/Max down to SQL instead).
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

// searchPage runs a repository Search fetching one extra row to derive TotalRecordCount, since the
// Search API returns no match count and CountAll can't see the search term. offset+len(rows) is
// exact once matches end (and a growing lower bound before), so paging terminates at the last match.
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

func (api *Router) listAlbums(ctx context.Context, opts model.QueryOptions, parentId, artistId string, contributingOnly bool, genreIds []string, scopeIDs []int, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Album(ctx)
	filters := squirrel.And{}
	// For albums, ParentId (browse an artist) and AlbumArtistIds/ArtistIds both mean "this artist's
	// albums"; contributingArtistIds means "albums they only appear on" (Featured On).
	switch {
	case contributingOnly && artistId != "":
		filters = append(filters, filter.AlbumsByContributingArtistID(artistId).Filters)
	case firstNonEmpty(artistId, parentId) != "":
		filters = append(filters, filter.AlbumsByArtistID(firstNonEmpty(artistId, parentId)).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if len(genreIds) > 0 {
		filters = append(filters, filter.ByGenreID(genreIds))
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

func (api *Router) listSongs(ctx context.Context, opts model.QueryOptions, parentId, artistId string, genreIds []string, scopeIDs []int, search string, fav bool, fields dto.Fields) (dto.QueryResult, error) {
	toItem := func(mf model.MediaFile) dto.BaseItemDto { return dto.SongToBaseItem(mf, fields) }
	repo := api.ds.MediaFile(ctx)
	filters := squirrel.And{}
	// For songs, ArtistIds/AlbumArtistIds selects an artist's tracks; ParentId selects an album's.
	switch {
	case artistId != "":
		filters = append(filters, filter.SongsByArtistID(artistId).Filters)
	case parentId != "":
		filters = append(filters, filter.SongsByAlbum(parentId).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if len(genreIds) > 0 {
		filters = append(filters, filter.ByGenreID(genreIds))
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
		return result(slice.Map(mfs, toItem), total, opts.Offset), nil
	}
	// When browsing an album's tracks, default to disc+track order (like Subsonic's GetAlbum); an
	// explicit client SortBy still wins, since applySort already set opts.Sort.
	if artistId == "" && parentId != "" && opts.Sort == "" {
		opts.Sort = filter.SongsByAlbum(parentId).Sort
	}
	mfs, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	return result(slice.Map(mfs, toItem), int(total), opts.Offset), nil
}

// listArtists lists artists in the given role: RoleAlbumArtist for the "album artists" views,
// RoleArtist for performing artists (/Artists). Without the role filter both lists would be identical.
// genreIds isn't applied to search — a name lookup, like role (see below).
func (api *Router) listArtists(ctx context.Context, opts model.QueryOptions, genreIds []string, scopeIDs []int, search string, fav bool, role model.Role) (dto.QueryResult, error) {
	repo := api.ds.Artist(ctx)

	// Artist Search does its own library scoping: it consumes a sole Eq{"library_id": ...} filter as a
	// search scope (artists have no library_id column). A compound or join-based filter
	// (ApplyArtistLibraryFilter) would leak into the FTS query and 500, so search and browse build
	// filters differently. Role isn't applied to search for the same reason — it's a name lookup.
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
	if len(genreIds) > 0 {
		opts.Filters = squirrel.And{opts.Filters, filter.ArtistsByGenreID(genreIds)}
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

// listGenres is intentionally unscoped: genres are global tags, not per-library entities. Paging is
// in-memory (GenreRepository has no CountAll, lists are small) so TotalRecordCount is the real total.
func (api *Router) listGenres(ctx context.Context, opts model.QueryOptions) (dto.QueryResult, error) {
	genres, err := api.ds.Genre(ctx).GetAll(model.QueryOptions{Sort: opts.Sort, Order: opts.Order})
	if err != nil {
		return dto.QueryResult{}, err
	}
	items := slice.Map(genres, dto.GenreToBaseItem)
	return result(paginate(items, opts.Offset, opts.Max), len(items), opts.Offset), nil
}

// listPlaylists lists playlists visible to the current user. Visibility (public or owned) is
// enforced by playlistRepository, not scopeIDs.
func (api *Router) listPlaylists(ctx context.Context, opts model.QueryOptions, favOnly bool) (dto.QueryResult, error) {
	if favOnly {
		starred := squirrel.Eq{"starred": true}
		if opts.Filters == nil {
			opts.Filters = starred
		} else {
			opts.Filters = squirrel.And{opts.Filters, starred}
		}
	}
	repo := api.ds.Playlist(ctx)
	playlists, err := repo.GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	total, err := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	if err != nil {
		return dto.QueryResult{}, err
	}
	return result(slice.Map(playlists, dto.PlaylistToBaseItem), int(total), opts.Offset), nil
}

// resolveItemByID resolves a decoded navidrome id to its BaseItemDto, trying library view, album,
// artist, song and playlist in turn. Albums and songs report not-found when the user lacks access
// to their library, so an id can't probe content outside the user's libraries.
func (api *Router) resolveItemByID(ctx context.Context, id string, fields dto.Fields) (dto.BaseItemDto, bool) {
	// The synthetic playlists folder must resolve by the id we advertised, not 404.
	if id == playlistsFolderID {
		return playlistsFolder(), true
	}
	u, _ := request.UserFrom(ctx)
	// Finamp resolves a /UserViews entry (Id=library id) by fetching it as a plain item; without this
	// the home screen and library tabs 404.
	if libID, err := strconv.Atoi(id); err == nil && u.HasLibraryAccess(libID) {
		for _, lib := range u.Libraries {
			if lib.ID == libID {
				return libraryView(lib), true
			}
		}
		// Admin bypass: Libraries is empty but all access is granted, so fetch the real library.
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
		// TODO: an artist spans multiple libraries (library_artist), so there's no single
		// LibraryID to gate here; artist access relies on list-time scoping and persistence.
		return dto.ArtistToBaseItem(*ar), true
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		if !u.HasLibraryAccess(mf.LibraryID) {
			return dto.BaseItemDto{}, false
		}
		return dto.SongToBaseItem(*mf, fields), true
	}
	// api.playlists.Get enforces ownership/visibility, so a non-owned or missing id falls through.
	if pl, err := api.playlists.Get(ctx, id); err == nil {
		return dto.PlaylistToBaseItem(*pl), true
	}
	return dto.BaseItemDto{}, false
}

// songsByIDs fetches the media files among ids with chunked IN queries instead of a Get per id.
func (api *Router) songsByIDs(ctx context.Context, ids []string) map[string]model.MediaFile {
	songs := make(map[string]model.MediaFile, len(ids))
	// Chunked to stay under SQLITE_MAX_VARIABLE_NUMBER, like playqueue's loadTracks.
	for chunk := range slice.CollectChunks(slices.Values(ids), 500) {
		mfs, err := api.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"media_file.id": chunk}})
		if err != nil {
			log.Error(ctx, "Jellyfin API: error fetching songs by id", err)
			continue
		}
		for _, mf := range mfs {
			songs[mf.ID] = mf
		}
	}
	return songs
}

// itemsByIDs resolves a decoded id list, keeping input order and skipping unresolvable ids.
// A Finamp-truncated id is resolved by prefix but echoed as requested — Finamp matches restored
// queue items against its stored (truncated) ids.
func (api *Router) itemsByIDs(ctx context.Context, ids []string, fields dto.Fields) dto.QueryResult {
	u, _ := request.UserFrom(ctx)
	fullIDs := api.resolveItemIDs(ctx, ids)
	songs := api.songsByIDs(ctx, fullIDs)
	var items []dto.BaseItemDto
	for i, id := range fullIDs {
		var item dto.BaseItemDto
		if mf, ok := songs[id]; ok {
			if !u.HasLibraryAccess(mf.LibraryID) {
				continue
			}
			item = dto.SongToBaseItem(mf, fields)
		} else if item, ok = api.resolveItemByID(ctx, id, fields); !ok {
			continue
		}
		if id != ids[i] {
			item.Id = dto.EncodeID(ids[i])
		}
		items = append(items, item)
	}
	return result(items, len(items), 0)
}

func (api *Router) getItem(w http.ResponseWriter, r *http.Request) {
	id := api.resolveItemID(r.Context(), dto.DecodeID(chi.URLParam(r, "itemId")))
	fields := dto.ParseFields(req.Params(r).StringOr("fields", ""))
	if item, ok := api.resolveItemByID(r.Context(), id, fields); ok {
		api.ok(w, r, item)
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

// deleteItem handles DELETE /Items/{id}. Only playlists are deletable here (albums/songs come from
// scanning), so a non-playlist id 404s. core/playlists.Delete enforces ownership.
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

// applySort translates Jellyfin's SortBy/SortOrder into a valid model.QueryOptions sort key for the
// item type. Clients send SortBy as a comma-separated fallback list (e.g. "DateCreated,SortName");
// this uses the first recognized key. An unrecognized SortBy is left untouched (the repo's default),
// not passed through raw where it could produce an invalid ORDER BY.
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

// sortColumnsByType maps lowercased-SortBy -> repo-sort-key per item type. Each repository maps
// logical fields to different real columns (e.g. media_file has "title" not "name"; artist has no
// "random").
var sortColumnsByType = map[string]map[string]string{
	"Audio": {
		"sortname": "title", "name": "title",
		"album": "album",
		// Finamp's album view sorts by ParentIndexNumber,IndexNumber (disc, track); Navidrome's
		// "album" sort key is disc+track order within an album, so map both to it.
		"indexnumber":       "album",
		"parentindexnumber": "album",
		"artist":            "artist",
		"albumartist":       "album_artist",
		"datecreated":       "recently_added",
		"playcount":         "play_count",
		"dateplayed":        "play_date",
		"communityrating":   "rating",
		"random":            "random",
		// Finamp's "Latest Releases" sorts by PremiereDate; "year" matches songs' ProductionYear.
		"premieredate":   "year",
		"productionyear": "year",
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
