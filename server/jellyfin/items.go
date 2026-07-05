package jellyfin

import (
	"context"
	"net/http"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/model"
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	api.ok(w, r, res)
}

// queryItems is the universal /Items dispatcher: it picks a target entity from
// IncludeItemTypes (falling back to MusicAlbum) and delegates to the matching listXxx.
func (api *Router) queryItems(ctx context.Context, r *http.Request) (dto.QueryResult, error) {
	p := req.Params(r)
	parentId := p.StringOr("ParentId", "")
	search := p.StringOr("SearchTerm", "")
	favOnly := strings.Contains(p.StringOr("Filters", ""), "IsFavorite")
	itemType := firstType(p.StringOr("IncludeItemTypes", ""))

	opts := model.QueryOptions{Offset: p.IntOr("StartIndex", 0), Max: p.IntOr("Limit", 0)}
	applySort(&opts, itemType, p.StringOr("SortBy", ""), p.StringOr("SortOrder", ""))

	switch itemType {
	case "Audio":
		return api.listSongs(ctx, opts, parentId, search, favOnly)
	case "MusicArtist":
		return api.listArtists(ctx, opts, search, favOnly)
	case "MusicGenre":
		return api.listGenres(ctx, opts)
	default: // MusicAlbum
		return api.listAlbums(ctx, opts, parentId, search, favOnly)
	}
}

// firstType picks the first recognized entry in the (possibly comma-separated)
// IncludeItemTypes, defaulting to MusicAlbum otherwise. This makes ParentId=<artistId>
// (no explicit type) browse into that artist's albums, matching the UserViews ->
// artists -> albums -> songs hierarchy; callers browsing into an album are expected to
// pass IncludeItemTypes=Audio explicitly, as Finamp and Jellyfin's own clients do.
func firstType(types string) string {
	for t := range strings.SplitSeq(types, ",") {
		t = strings.TrimSpace(t)
		switch t {
		case "Audio", "MusicArtist", "MusicAlbum", "MusicGenre":
			return t
		}
	}
	return "MusicAlbum"
}

func (api *Router) listAlbums(ctx context.Context, opts model.QueryOptions, parentId, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Album(ctx)
	filters := squirrel.And{}
	if parentId != "" && parentId != musicViewID {
		filters = append(filters, filter.AlbumsByArtistID(parentId).Filters)
	} else {
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters

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

func (api *Router) listSongs(ctx context.Context, opts model.QueryOptions, parentId, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.MediaFile(ctx)
	filters := squirrel.And{}
	if parentId != "" && parentId != musicViewID {
		filters = append(filters, filter.SongsByAlbum(parentId).Filters)
	} else {
		filters = append(filters, notMissing)
	}
	if fav {
		filters = append(filters, filter.ByStarred().Filters)
	}
	opts.Filters = filters

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

func (api *Router) listArtists(ctx context.Context, opts model.QueryOptions, search string, fav bool) (dto.QueryResult, error) {
	repo := api.ds.Artist(ctx)
	if fav {
		opts.Filters = filter.ArtistsByStarred().Filters
	} else {
		opts.Filters = notMissing
	}

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

func (api *Router) listGenres(ctx context.Context, opts model.QueryOptions) (dto.QueryResult, error) {
	genres, err := api.ds.Genre(ctx).GetAll(opts)
	if err != nil {
		return dto.QueryResult{}, err
	}
	return result(slice.Map(genres, dto.GenreToBaseItem), len(genres), opts.Offset), nil
}

func (api *Router) getItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "itemId")
	if al, err := api.ds.Album(ctx).Get(id); err == nil {
		api.ok(w, r, dto.AlbumToBaseItem(*al))
		return
	}
	if ar, err := api.ds.Artist(ctx).Get(id); err == nil {
		api.ok(w, r, dto.ArtistToBaseItem(*ar))
		return
	}
	if mf, err := api.ds.MediaFile(ctx).Get(id); err == nil {
		api.ok(w, r, dto.SongToBaseItem(*mf))
		return
	}
	http.Error(w, "Not Found", http.StatusNotFound)
}

func (api *Router) getLatest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	opts := filter.AlbumsByNewest()
	opts.Max = req.Params(r).IntOr("Limit", 20)
	albums, err := api.ds.Album(ctx).GetAll(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

// applySort translates Jellyfin's SortBy/SortOrder into a model.QueryOptions sort key valid
// for the given item type. Each repository maps different logical fields to different (or
// annotation-joined) real columns, so a key valid for one type can be an invalid column for
// another (e.g. media_file has no "name" column, only "title"; artist has no "random" column).
// An unrecognized SortBy is dropped rather than passed through raw, to avoid an invalid ORDER BY.
func applySort(opts *model.QueryOptions, itemType, sortBy, order string) {
	switch itemType {
	case "Audio":
		switch strings.ToLower(sortBy) {
		case "sortname", "name":
			opts.Sort = "title"
		case "album":
			opts.Sort = "album"
		case "albumartist":
			opts.Sort = "album_artist"
		case "datecreated":
			opts.Sort = "recently_added"
		case "playcount":
			opts.Sort = "play_count"
		case "dateplayed":
			opts.Sort = "play_date"
		case "random":
			opts.Sort = "random"
		}
	case "MusicArtist":
		switch strings.ToLower(sortBy) {
		case "sortname", "name":
			opts.Sort = "name"
		case "albumcount":
			opts.Sort = "album_count"
		case "songcount":
			opts.Sort = "song_count"
		case "datecreated":
			opts.Sort = "created_at"
		case "playcount":
			opts.Sort = "play_count"
		case "dateplayed":
			opts.Sort = "play_date"
		}
	default: // MusicAlbum
		switch strings.ToLower(sortBy) {
		case "sortname", "name", "album":
			opts.Sort = "name"
		case "albumartist":
			opts.Sort = "album_artist"
		case "datecreated":
			opts.Sort = "recently_added"
		case "random":
			opts.Sort = "random"
		case "playcount":
			opts.Sort = "play_count"
		case "dateplayed":
			opts.Sort = "play_date"
		case "premieredate", "productionyear":
			opts.Sort = "max_year"
		}
	}
	if strings.EqualFold(order, "Descending") {
		opts.Order = "desc"
	}
}
