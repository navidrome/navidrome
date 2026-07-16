package jellyfin

import (
	"context"
	"io"
	"iter"
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

// searchTerm trims, so a whitespace-only term is not a search: doSearch would read it as "match
// everything" and materialize the library, where the unfiltered path streams.
func searchTerm(p *req.Values) string {
	return strings.TrimSpace(p.StringOr("searchterm", ""))
}

func (api *Router) getItems(w http.ResponseWriter, r *http.Request) {
	res, err := api.queryItems(r.Context(), r)
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	api.ok(w, r, res)
}

// itemsResult is the outcome of a collection query: a materialized page, or a cursor opener so a
// full-library response never builds every DTO at once. Exactly one of items/openCursor is set.
//
// openCursor is deferred rather than opened here: it must run after the ServerId lookup, which
// writes to the DB on first use and would deadlock against an open reader, but before the first
// response byte, so a failed open is still a clean error rather than a truncated 200.
type itemsResult struct {
	items      []dto.BaseItemDto
	openCursor func() (iter.Seq2[dto.BaseItemDto, error], error)
	total      int
	start      int
}

func materialized(q dto.QueryResult) itemsResult {
	return itemsResult{items: q.Items, total: q.TotalRecordCount, start: q.StartIndex}
}

func streamed(open func() (iter.Seq2[dto.BaseItemDto, error], error), total, start int) itemsResult {
	return itemsResult{openCursor: open, total: total, start: start}
}

// chained streams several results back to back, skipping the first skip items — the unbounded
// multi-type merge, where paginate(items, offset, 0) is just the concatenation minus its head.
func chained(results []itemsResult, total, skip int) itemsResult {
	open := func() (iter.Seq2[dto.BaseItemDto, error], error) {
		if len(results) == 0 {
			return sliceItems(nil), nil
		}
		// Only the first opens eagerly (so the usual failure is still a clean error); the rest open as
		// the stream reaches them, so only one cursor pins a DB connection at a time.
		first, err := results[0].seq()
		if err != nil {
			return nil, err
		}
		return func(yield func(dto.BaseItemDto, error) bool) {
			n := 0
			emit := func(seq iter.Seq2[dto.BaseItemDto, error]) bool {
				for it, err := range seq {
					if err != nil {
						yield(dto.BaseItemDto{}, err)
						return false
					}
					if n < skip {
						n++
						continue
					}
					if !yield(it, nil) {
						return false
					}
				}
				return true
			}
			if !emit(first) {
				return
			}
			for _, res := range results[1:] {
				seq, err := res.seq()
				if err != nil {
					yield(dto.BaseItemDto{}, err)
					return
				}
				if !emit(seq) {
					return
				}
			}
		}, nil
	}
	return streamed(open, total, skip)
}

// streamCursor builds a deferred opener that maps each row as it's yielded. It takes the cursor's
// underlying func type, so callers wrap repo.GetCursor for the named type to infer T.
func streamCursor[T any](openCursor func() (func(func(T, error) bool), error), toItem func(T) dto.BaseItemDto) func() (iter.Seq2[dto.BaseItemDto, error], error) {
	return func() (iter.Seq2[dto.BaseItemDto, error], error) {
		cursor, err := openCursor()
		if err != nil {
			return nil, err
		}
		return func(yield func(dto.BaseItemDto, error) bool) {
			for row, err := range cursor {
				if err != nil {
					yield(dto.BaseItemDto{}, err)
					return
				}
				if !yield(toItem(row), nil) {
					return
				}
			}
		}, nil
	}
}

// seq returns the items as one sequence, opening the cursor if there is one.
func (ir itemsResult) seq() (iter.Seq2[dto.BaseItemDto, error], error) {
	if ir.openCursor != nil {
		return ir.openCursor()
	}
	return sliceItems(ir.items), nil
}

// collect drains the result into a slice, for the merge that combines types before paginating.
func (ir itemsResult) collect() ([]dto.BaseItemDto, error) {
	if ir.openCursor == nil {
		return ir.items, nil
	}
	seq, err := ir.openCursor()
	if err != nil {
		return nil, err
	}
	var out []dto.BaseItemDto
	for it, err := range seq {
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, nil
}

func (api *Router) writeItems(w http.ResponseWriter, r *http.Request, res itemsResult) {
	api.streamResult(w, r, res, func(w io.Writer, items iter.Seq2[dto.BaseItemDto, error]) error {
		return streamItemsEnvelope(w, items, res.total, res.start)
	})
}

// writeItemsArray writes the bare-array shape (/Items/Latest), which has no QueryResult envelope.
func (api *Router) writeItemsArray(w http.ResponseWriter, r *http.Request, res itemsResult) {
	api.streamResult(w, r, res, streamItemsArray)
}

// streamResult stamps every item's ServerId (constant per request, so it's set here rather than in
// each mapper). The cursor opens before the first byte, so a failed open is still a clean 500.
func (api *Router) streamResult(w http.ResponseWriter, r *http.Request, res itemsResult,
	write func(io.Writer, iter.Seq2[dto.BaseItemDto, error]) error) {
	sid := api.serverID(r.Context())
	seq, err := res.seq()
	if err != nil {
		api.internalError(w, r, err)
		return
	}
	stamped := func(yield func(dto.BaseItemDto, error) bool) {
		for it, err := range seq {
			if err != nil {
				yield(dto.BaseItemDto{}, err)
				return
			}
			it.ServerId = sid
			if !yield(it, nil) {
				return
			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := write(w, stamped); err != nil {
		log.Error(r.Context(), "Jellyfin API: error streaming response", err)
	}
}

// itemsQuery is a parsed /Items request, so the dispatch and every listXxx take one value instead
// of a long positional parameter list.
type itemsQuery struct {
	fields    dto.Fields
	ids       []string
	rawTypes  string
	types     []string
	search    string
	sortBy    string
	sortOrder string
	offset    int
	limit     int
	favOnly   bool
	// parentId scopes the query. entityParent is the same id only when it names an entity (an artist
	// for MusicAlbum, an album for Audio) rather than a library.
	parentId        string
	entityParent    string
	isLibraryParent bool
	scopeIDs        []int
	// artistId selects that artist's own discography; contributingOnly means albums they merely
	// appear on (Jellyfin's "Featured On"), which must exclude that discography.
	artistId         string
	contributingOnly bool
	genreIds         []string
	albumIds         []string
}

// parseItemsQuery also resolves the entity types (inferring them from the parent when
// IncludeItemTypes is absent) and the library scope. Query keys are read lowercase because
// normalizeQueryKeys folded them (Jellyfin binds case-insensitively).
func (api *Router) parseItemsQuery(ctx context.Context, r *http.Request) itemsQuery {
	p := req.Params(r)
	q := itemsQuery{
		fields:    dto.ParseFields(p.StringOr("fields", "")),
		ids:       decodedQueryIDs(r, "ids"),
		rawTypes:  p.StringOr("includeitemtypes", ""),
		search:    searchTerm(p),
		sortBy:    p.StringOr("sortby", ""),
		sortOrder: p.StringOr("sortorder", ""),
		offset:    p.IntOr("startindex", 0),
		limit:     p.IntOr("limit", 0),
		// Clients express "favorites only" two ways: Filters=IsFavorite and the standalone
		// isFavorite=true param (Finamp's "Favourite tracks" widget uses the latter).
		favOnly:  strings.Contains(p.StringOr("filters", ""), "IsFavorite") || p.BoolOr("isfavorite", false),
		parentId: dto.DecodeID(p.StringOr("parentid", "")),
		// Finamp's genre screen sends ParentId=<libraryId> for scoping plus GenreIds for the genre.
		genreIds: decodedQueryIDs(r, "genreids"),
		// Feishin fetches an album's tracks with AlbumIds instead of ParentId.
		albumIds: decodedQueryIDs(r, "albumids"),
	}
	// An artist's page filters by artist, not ParentId: Finamp sends ParentId=<libraryId> for scoping
	// plus AlbumArtistIds/ArtistIds/contributingArtistIds for the artist.
	albumArtistScope := firstNonEmpty(p.StringOr("albumartistids", ""), p.StringOr("artistids", ""))
	contributingScope := p.StringOr("contributingartistids", "")
	q.artistId = firstDecodedID(firstNonEmpty(albumArtistScope, contributingScope))
	q.contributingOnly = albumArtistScope == "" && contributingScope != ""

	q.types = parseTypes(q.rawTypes)
	q.scopeIDs, q.isLibraryParent = resolveLibraryScope(ctx, q.parentId)

	// Recursive=false asks for direct children only, and no track is a library's direct child.
	// Finamp's sync probes a library this way, and every track is a wrong, unbounded answer.
	if q.isLibraryParent && !p.BoolOr("recursive", false) {
		q.types = slices.DeleteFunc(q.types, func(t string) bool { return t == "Audio" })
	}

	// With no item type, Jellyfin infers the child type from the parent: album parent -> its tracks
	// (Jellify opens albums this way). An artist parent keeps parseTypes' MusicAlbum default (browse
	// its albums).
	if q.rawTypes == "" && q.parentId != "" && !q.isLibraryParent {
		if q.parentId == playlistsFolderID {
			// Browsing into the synthetic playlists folder lists the user's playlists.
			q.types = []string{"Playlist"}
		} else if _, err := api.ds.Album(ctx).Get(q.parentId); err == nil {
			q.types = []string{"Audio"}
		}
	}
	// ParentId-as-entity-id only makes sense for a single type; a multi-type query has no natural
	// parent entity, so there ParentId is only library scoping.
	q.entityParent = q.parentId
	if q.isLibraryParent || len(q.types) > 1 {
		q.entityParent = ""
	}
	return q
}

// queryItems is the /Items dispatcher: it resolves the request to entity types and queries each via
// the matching listXxx, merging multi-type results into one paginated list (as Finamp's favorites
// screen requests).
func (api *Router) queryItems(ctx context.Context, r *http.Request) (itemsResult, error) {
	q := api.parseItemsQuery(ctx, r)
	switch {
	// /Items?ids= is a batch-fetch-by-id that bypasses the type dispatch.
	case len(q.ids) > 0:
		return materialized(api.itemsByIDs(ctx, q.ids, q.fields)), nil
	// A ManualPlaylistsFolder query asks for the synthetic "playlists library" container, not real items.
	case strings.Contains(q.rawTypes, "ManualPlaylistsFolder"):
		return materialized(result([]dto.BaseItemDto{playlistsFolder()}, 1, 0)), nil
	}
	if repo, ok := api.playlistTracksRepo(ctx, q); ok {
		return api.playlistTrackPage(repo, q.fields, q.offset, q.limit)
	}
	if q.search != "" {
		q.limit = clampLimit(q.limit, defaultSearchLimit, maxSearchLimit)
	}
	if len(q.types) == 1 {
		opts := model.QueryOptions{Offset: q.offset, Max: q.limit}
		applySort(&opts, q.types[0], q.sortBy, q.sortOrder)
		return api.queryItemsOfType(ctx, q.types[0], opts, q)
	}
	return api.mergeTypes(ctx, q)
}

// playlistTracksRepo resolves a playlist parent, whatever IncludeItemTypes says: Jellify opens a
// playlist with ParentId=<playlist>&IncludeItemTypes=Audio, and routing that through listSongs would
// treat the playlist id as an album id and return nothing.
//
// ok is false when ParentId isn't a visible playlist, so the caller falls through to the type
// dispatch: ParentId is usually an album or artist.
func (api *Router) playlistTracksRepo(ctx context.Context, q itemsQuery) (model.PlaylistTrackRepository, bool) {
	if q.parentId == "" || q.isLibraryParent || q.parentId == playlistsFolderID {
		return nil, false
	}
	// Tracks enforces visibility.
	repo, err := api.playlists.Tracks(ctx, q.parentId)
	return repo, err == nil
}

func (api *Router) mergeTypes(ctx context.Context, q itemsQuery) (itemsResult, error) {
	// Each per-type query needs at most offset+limit rows (the worst case where one type fills the
	// whole [offset, offset+limit) window). Totals are unaffected — they come from CountAll.
	window := 0
	if q.limit > 0 {
		window = q.offset + q.limit
	}
	// A search can't stream, so the window is what each type materializes and StartIndex would drive
	// it without bound. Only below the window are the merged rows the true order, hence the clip
	// below too. Non-search stays unbounded in StartIndex: a known gap, fixable with per-type counts.
	if q.search != "" {
		window = min(window, maxSearchLimit)
	}
	var results []itemsResult
	total := 0
	for _, itemType := range q.types {
		var opts model.QueryOptions
		opts.Max = window
		applySort(&opts, itemType, q.sortBy, q.sortOrder)
		res, err := api.queryItemsOfType(ctx, itemType, opts, q)
		if err != nil {
			return itemsResult{}, err
		}
		results = append(results, res)
		total += res.total
	}
	if q.limit == 0 {
		// No cap above, so merging in memory would pull every row of every type. The merged page is
		// just their rows in order minus the first offset — what chaining the cursors yields.
		return chained(results, total, q.offset), nil
	}
	var items []dto.BaseItemDto
	for _, res := range results {
		typeItems, err := res.collect()
		if err != nil {
			return itemsResult{}, err
		}
		items = append(items, typeItems...)
	}
	if q.search != "" {
		// Past the window the merged order isn't the true one, so drop it rather than serve another
		// type's rows. The total is what's pageable overall, not this page, or a client paging on it
		// would stop after the first page.
		items = items[:min(window, len(items))]
		total = min(total, maxSearchLimit)
	}
	return materialized(result(paginate(items, q.offset, q.limit), total, q.offset)), nil
}

func (api *Router) queryItemsOfType(ctx context.Context, itemType string, opts model.QueryOptions, q itemsQuery) (itemsResult, error) {
	switch itemType {
	case "Audio":
		return api.listSongs(ctx, opts, q)
	case "MusicArtist":
		// The MusicArtist browse hierarchy (UserViews -> artists -> albums) means album artists.
		return api.listArtists(ctx, opts, q, model.RoleAlbumArtist)
	case "MusicGenre":
		return api.listGenres(ctx, opts)
	case "Playlist":
		return api.listPlaylists(ctx, opts, q)
	default: // MusicAlbum
		return api.listAlbums(ctx, opts, q)
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

// Search can't stream (Search returns a slice), so it needs both a default and a ceiling: without
// the ceiling, Limit=999999 still materializes every match.
const (
	defaultSearchLimit = 100
	maxSearchLimit     = 2000
)

// clampLimit bounds a client-supplied limit, 0 or less meaning it sent none, so it can't drive an
// oversized allocation or provider fetch (flagged by CodeQL as a user-controlled allocation size).
//
// Searches clamp their Limit here rather than in searchPage, which also sees mergeTypes' larger
// offset+limit window: bounding that would truncate each type before the merged page is cut.
func clampLimit(limit, def, ceiling int) int {
	if limit <= 0 {
		return def
	}
	return min(limit, ceiling)
}

// searchPage runs a repository Search fetching one extra row to derive TotalRecordCount, since the
// Search API returns no match count and CountAll can't see the search term. offset+len(rows) is
// exact once matches end (and a growing lower bound before), so paging terminates at the last match.
func searchPage[S ~[]E, E any](opts model.QueryOptions, search func(model.QueryOptions) (S, error)) (S, int, error) {
	fetch := opts
	fetch.Max++
	rows, err := search(fetch)
	if err != nil {
		return nil, 0, err
	}
	total := opts.Offset + len(rows)
	if len(rows) > opts.Max {
		rows = rows[:opts.Max]
	}
	return rows, total, nil
}

func (api *Router) listAlbums(ctx context.Context, opts model.QueryOptions, q itemsQuery) (itemsResult, error) {
	repo := api.ds.Album(ctx)
	filters := squirrel.And{}
	// For albums, ParentId (browse an artist) and AlbumArtistIds/ArtistIds both mean "this artist's
	// albums"; contributingArtistIds means "albums they only appear on" (Featured On).
	switch {
	case q.contributingOnly && q.artistId != "":
		filters = append(filters, filter.AlbumsByContributingArtistID(q.artistId).Filters)
	case firstNonEmpty(q.artistId, q.entityParent) != "":
		filters = append(filters, filter.AlbumsByArtistID(firstNonEmpty(q.artistId, q.entityParent)).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if len(q.genreIds) > 0 {
		filters = append(filters, filter.ByGenreID(q.genreIds))
	}
	if q.favOnly {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, q.scopeIDs)

	if q.search != "" {
		albums, total, err := searchPage(opts, func(o model.QueryOptions) (model.Albums, error) {
			return repo.Search(q.search, o)
		})
		if err != nil {
			return itemsResult{}, err
		}
		return materialized(result(slice.Map(albums, dto.AlbumToBaseItem), total, opts.Offset)), nil
	}
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	open := streamCursor(func() (func(func(model.Album, error) bool), error) {
		return repo.GetCursor(opts)
	}, dto.AlbumToBaseItem)
	return streamed(open, int(total), opts.Offset), nil
}

func (api *Router) listSongs(ctx context.Context, opts model.QueryOptions, q itemsQuery) (itemsResult, error) {
	toItem := func(mf model.MediaFile) dto.BaseItemDto { return dto.SongToBaseItem(mf, q.fields) }
	repo := api.ds.MediaFile(ctx)
	filters := squirrel.And{}
	// For songs, ArtistIds/AlbumArtistIds selects an artist's tracks; ParentId selects an album's.
	switch {
	case q.artistId != "":
		filters = append(filters, filter.SongsByArtistID(q.artistId).Filters)
	case q.entityParent != "":
		filters = append(filters, filter.SongsByAlbum(q.entityParent).Filters)
	default:
		filters = append(filters, notMissing)
	}
	if len(q.albumIds) > 0 {
		filters = append(filters, filter.ByAlbumID(q.albumIds))
	}
	if len(q.genreIds) > 0 {
		filters = append(filters, filter.ByGenreID(q.genreIds))
	}
	if q.favOnly {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters
	opts = filter.ApplyLibraryFilter(opts, q.scopeIDs)

	if q.search != "" {
		mfs, total, err := searchPage(opts, func(o model.QueryOptions) (model.MediaFiles, error) {
			return repo.Search(q.search, o)
		})
		if err != nil {
			return itemsResult{}, err
		}
		return materialized(result(slice.Map(mfs, toItem), total, opts.Offset)), nil
	}
	// When browsing an album's tracks, default to disc+track order (like Subsonic's GetAlbum); an
	// explicit client SortBy still wins, since applySort already set opts.Sort.
	if q.artistId == "" && q.entityParent != "" && opts.Sort == "" {
		opts.Sort = filter.SongsByAlbum(q.entityParent).Sort
	}
	// A full-library request (Finamp's sync, with MediaSources) is tens of thousands of fat rows.
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	open := streamCursor(func() (func(func(model.MediaFile, error) bool), error) {
		return repo.GetCursor(opts)
	}, toItem)
	return streamed(open, int(total), opts.Offset), nil
}

// listArtists lists artists in the given role: RoleAlbumArtist for the "album artists" views,
// RoleArtist for performing artists (/Artists). Without the role filter both lists would be identical.
// genreIds isn't applied to search — a name lookup, like role (see below).
func (api *Router) listArtists(ctx context.Context, opts model.QueryOptions, q itemsQuery, role model.Role) (itemsResult, error) {
	repo := api.ds.Artist(ctx)

	// Artist Search does its own library scoping: it consumes a sole Eq{"library_id": ...} filter as a
	// search scope (artists have no library_id column). A compound or join-based filter
	// (ApplyArtistLibraryFilter) would leak into the FTS query and 500, so search and browse build
	// filters differently. Role isn't applied to search for the same reason — it's a name lookup.
	if q.search != "" {
		if len(q.scopeIDs) > 0 {
			opts.Filters = squirrel.Eq{"library_id": q.scopeIDs}
		}
		artists, total, err := searchPage(opts, func(o model.QueryOptions) (model.Artists, error) {
			return repo.Search(q.search, o)
		})
		if err != nil {
			return itemsResult{}, err
		}
		return materialized(result(slice.Map(artists, dto.ArtistToBaseItem), total, opts.Offset)), nil
	}

	if q.favOnly {
		opts.Filters = filter.ArtistsByStarred().Filters
	} else {
		opts.Filters = notMissing
	}
	if len(q.genreIds) > 0 {
		opts.Filters = squirrel.And{opts.Filters, filter.ArtistsByGenreID(q.genreIds)}
	}
	opts = filter.ArtistsByRole(opts, role)
	opts = filter.ApplyArtistLibraryFilter(opts, q.scopeIDs)
	total, _ := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	open := streamCursor(func() (func(func(model.Artist, error) bool), error) {
		return repo.GetCursor(opts)
	}, dto.ArtistToBaseItem)
	return streamed(open, int(total), opts.Offset), nil
}

// listGenres is intentionally unscoped: genres are global tags, not per-library entities. It's also
// the one listXxx that stays materialized: GenreRepository has no CountAll, so the total is the
// length of the full list and paging is in-memory — nothing for a cursor to page over.
func (api *Router) listGenres(ctx context.Context, opts model.QueryOptions) (itemsResult, error) {
	genres, err := api.ds.Genre(ctx).GetAll(model.QueryOptions{Sort: opts.Sort, Order: opts.Order})
	if err != nil {
		return itemsResult{}, err
	}
	items := slice.Map(genres, dto.GenreToBaseItem)
	return materialized(result(paginate(items, opts.Offset, opts.Max), len(items), opts.Offset)), nil
}

// listPlaylists lists playlists visible to the current user. Visibility (public or owned) is
// enforced by playlistRepository, not scopeIDs.
func (api *Router) listPlaylists(ctx context.Context, opts model.QueryOptions, q itemsQuery) (itemsResult, error) {
	if q.favOnly {
		starred := squirrel.Eq{"starred": true}
		if opts.Filters == nil {
			opts.Filters = starred
		} else {
			opts.Filters = squirrel.And{opts.Filters, starred}
		}
	}
	repo := api.ds.Playlist(ctx)
	total, err := repo.CountAll(model.QueryOptions{Filters: opts.Filters})
	if err != nil {
		return itemsResult{}, err
	}
	open := streamCursor(func() (func(func(model.Playlist, error) bool), error) {
		return repo.GetCursor(opts)
	}, dto.PlaylistToBaseItem)
	return streamed(open, int(total), opts.Offset), nil
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

// getLatest returns a bare array, not a QueryResult envelope — real Jellyfin's shape for
// /Items/Latest, and why it writes directly instead of going through api.ok.
func (api *Router) getLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	opts := filter.AlbumsByNewest()
	opts.Max = req.Params(r).IntOr("limit", 20)
	opts = filter.ApplyLibraryFilter(opts, accessibleLibraryIDs(ctx))
	repo := api.ds.Album(ctx)
	open := streamCursor(func() (func(func(model.Album, error) bool), error) {
		return repo.GetCursor(opts)
	}, dto.AlbumToBaseItem)
	api.writeItemsArray(w, r, streamed(open, 0, 0))
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
