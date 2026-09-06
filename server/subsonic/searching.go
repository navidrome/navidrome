package subsonic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/deluan/sanitize"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/adapters/rustsearch"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/query"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"golang.org/x/sync/errgroup"
)

type searchParams struct {
	query        string
	artistCount  int
	artistOffset int
	albumCount   int
	albumOffset  int
	songCount    int
	songOffset   int
}

const maxSearchQueryRunes = 256

func (api *Router) getSearchParams(r *http.Request) (*searchParams, error) {
	p := req.Params(r)
	sp := &searchParams{}
	sp.query = p.StringOr("query", `""`)
	if utf8.RuneCountInString(sp.query) > maxSearchQueryRunes {
		return nil, fmt.Errorf("search query exceeds %d characters", maxSearchQueryRunes)
	}
	sp.artistCount = normalizeSearchCount(p.IntOr("artistCount", 20))
	sp.artistOffset = max(p.IntOr("artistOffset", 0), 0)
	sp.albumCount = normalizeSearchCount(p.IntOr("albumCount", 20))
	sp.albumOffset = max(p.IntOr("albumOffset", 0), 0)
	sp.songCount = normalizeSearchCount(p.IntOr("songCount", 20))
	sp.songOffset = max(p.IntOr("songOffset", 0), 0)
	return sp, nil
}

func normalizeSearchCount(count int) int {
	return min(max(count, 0), rustsearch.MaxResults)
}

type searchFunc[T any] func(q string, options ...model.QueryOptions) (T, error)

func callSearch[T any](ctx context.Context, s searchFunc[T], q string, options model.QueryOptions, result *T) func() error {
	return func() error {
		if options.Max == 0 {
			return nil
		}
		var err error
		if log.IsGreaterOrEqualTo(log.LevelTrace) {
			typ := strings.TrimPrefix(reflect.TypeOf(*result).String(), "model.")
			start := time.Now()
			*result, err = s(q, options)
			if err != nil {
				logSearchFailure(ctx, "Error searching "+typ, q, time.Since(start), err)
				return err
			}
			log.Trace(ctx, "Search for "+typ+" completed", "query", q, "elapsed", time.Since(start))
			return nil
		}
		*result, err = s(q, options)
		if err != nil {
			logSearchFailure(ctx, "Error searching", q, 0, err)
			return err
		}
		return nil
	}
}

func logSearchFailure(ctx context.Context, message, query string, elapsed time.Duration, err error) {
	args := make([]any, 0, 7)
	args = append(args, ctx, message, "query", query)
	if elapsed > 0 {
		args = append(args, "elapsed", elapsed)
	}
	args = append(args, err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		args[1] = "Search request canceled"
		log.Debug(args...)
		return
	}
	log.Error(args...)
}

func (api *Router) searchAll(ctx context.Context, sp *searchParams, musicFolderIds []int) (mediaFiles model.MediaFiles, albums model.Albums, artists model.Artists) {
	start := time.Now()
	q := sanitize.Accents(strings.ToLower(strings.TrimSuffix(sp.query, "*")))
	if api.rustSearch != nil && rustSearchableQuery(q) && rustSearchPageSupported(sp) {
		scope := musicFolderIds
		if len(scope) == 0 {
			scope = getUserAccessibleLibraries(ctx).IDs()
		}
		if mfs, als, as, ok := api.searchAllRust(ctx, q, scope, sp); ok {
			if log.IsGreaterOrEqualTo(log.LevelDebug) {
				log.Debug(ctx, "Search completed", "backend", "rust", "songs", len(mfs), "albums", len(als),
					"artists", len(as), "query", sp.query, "elapsedTime", time.Since(start))
			}
			return mfs, als, as
		}
	}

	// Build options with offset/size/filters packed in
	songOpts := model.QueryOptions{Max: sp.songCount, Offset: sp.songOffset}
	albumOpts := model.QueryOptions{Max: sp.albumCount, Offset: sp.albumOffset}
	artistOpts := model.QueryOptions{Max: sp.artistCount, Offset: sp.artistOffset}

	if len(musicFolderIds) > 0 {
		songOpts.Filters = query.Eq("library_id", musicFolderIds)
		albumOpts.Filters = query.Eq("library_id", musicFolderIds)
		artistOpts.Filters = query.Eq("library_id", musicFolderIds)
	}

	// Run searches in parallel
	g, ctx := errgroup.WithContext(ctx)
	g.Go(callSearch(ctx, api.ds.MediaFile(ctx).Search, q, songOpts, &mediaFiles))
	g.Go(callSearch(ctx, api.ds.Album(ctx).Search, q, albumOpts, &albums))
	g.Go(callSearch(ctx, api.ds.Artist(ctx).Search, q, artistOpts, &artists))
	err := g.Wait()
	if err == nil {
		if log.IsGreaterOrEqualTo(log.LevelDebug) {
			log.Debug(ctx, "Search completed", "backend", "sqlite", "songs", len(mediaFiles), "albums", len(albums), "artists", len(artists),
				"query", sp.query, "elapsedTime", time.Since(start))
		}
	} else {
		log.Warn(ctx, "Search was interrupted", "query", sp.query, "elapsedTime", time.Since(start), err)
	}
	return mediaFiles, albums, artists
}

func rustSearchPageSupported(sp *searchParams) bool {
	return searchPageInWindow(sp.songOffset, sp.songCount) &&
		searchPageInWindow(sp.albumOffset, sp.albumCount) &&
		searchPageInWindow(sp.artistOffset, sp.artistCount)
}

func searchPageInWindow(offset, count int) bool {
	return offset >= 0 && count >= 0 && count <= rustsearch.MaxResults &&
		(count == 0 || offset <= rustsearch.MaxResults-count)
}

func rustSearchableQuery(query string) bool {
	trimmed := strings.Trim(strings.TrimSpace(query), `"`)
	if _, err := uuid.Parse(trimmed); err == nil {
		return false
	}
	searchable := 0
	for _, char := range query {
		if unicode.Is(unicode.Han, char) ||
			unicode.Is(unicode.Hiragana, char) ||
			unicode.Is(unicode.Katakana, char) ||
			unicode.Is(unicode.Hangul, char) {
			return true
		}
		if unicode.IsLetter(char) || unicode.IsNumber(char) {
			searchable++
			if searchable >= 2 {
				return true
			}
		}
	}
	return false
}

func (api *Router) searchAllRust(ctx context.Context, query string, libraryIDs []int, sp *searchParams) (model.MediaFiles, model.Albums, model.Artists, bool) {
	results, err := api.rustSearch.SearchAll(
		ctx,
		query,
		libraryIDs,
		rustsearch.SearchLimits{Offset: sp.songOffset, Limit: sp.songCount},
		rustsearch.SearchLimits{Offset: sp.albumOffset, Limit: sp.albumCount},
		rustsearch.SearchLimits{Offset: sp.artistOffset, Limit: sp.artistCount},
	)
	if err != nil {
		if !errors.Is(err, rustsearch.ErrNotReady) {
			log.Warn(ctx, "Rust grouped search failed; using SQLite fallback", err)
		}
		return nil, nil, nil, false
	}

	mediaFiles, albums, artists, err := api.hydrateRustSearchResults(ctx, results)
	if err != nil {
		log.Warn(ctx, "Hydrating grouped Rust search results failed; using SQLite fallback", err)
		return nil, nil, nil, false
	}
	if rustSearchResultsEmpty(sp, mediaFiles, albums, artists) {
		log.Debug(ctx, "Rust grouped search returned no hits for requested buckets; using SQLite fallback", "query", query)
		return nil, nil, nil, false
	}
	return mediaFiles, albums, artists, true
}

func rustSearchResultsEmpty(sp *searchParams, mediaFiles model.MediaFiles, albums model.Albums, artists model.Artists) bool {
	requested := sp.songCount > 0 || sp.albumCount > 0 || sp.artistCount > 0
	if !requested {
		return false
	}
	if sp.songCount > 0 && len(mediaFiles) > 0 {
		return false
	}
	if sp.albumCount > 0 && len(albums) > 0 {
		return false
	}
	if sp.artistCount > 0 && len(artists) > 0 {
		return false
	}
	return true
}

// hydrateRustSearchResults loads song/album/artist rows for Tantivy hit IDs.
// Empty buckets are skipped. When more than one bucket has hits, hydrates run
// in parallel; otherwise a single GetAll avoids errgroup overhead. Previously
// all three GetAlls always ran even when a Subsonic count was 0.
func (api *Router) hydrateRustSearchResults(ctx context.Context, results rustsearch.SearchResults) (model.MediaFiles, model.Albums, model.Artists, error) {
	var mediaFiles model.MediaFiles
	var albums model.Albums
	var artists model.Artists

	type hydrateJob struct {
		run func(context.Context) error
	}
	jobs := make([]hydrateJob, 0, 3)
	if len(results.SongIDs) > 0 {
		jobs = append(jobs, hydrateJob{run: func(c context.Context) error {
			var err error
			mediaFiles, err = api.hydrateRustSongs(c, results.SongIDs)
			return err
		}})
	}
	if len(results.AlbumIDs) > 0 {
		jobs = append(jobs, hydrateJob{run: func(c context.Context) error {
			var err error
			albums, err = api.hydrateRustAlbums(c, results.AlbumIDs)
			return err
		}})
	}
	if len(results.ArtistIDs) > 0 {
		jobs = append(jobs, hydrateJob{run: func(c context.Context) error {
			var err error
			artists, err = api.hydrateRustArtists(c, results.ArtistIDs)
			return err
		}})
	}
	switch len(jobs) {
	case 0:
		return mediaFiles, albums, artists, nil
	case 1:
		err := jobs[0].run(ctx)
		return mediaFiles, albums, artists, err
	default:
		g, hydrateCtx := errgroup.WithContext(ctx)
		for _, job := range jobs {
			g.Go(func() error { return job.run(hydrateCtx) })
		}
		err := g.Wait()
		return mediaFiles, albums, artists, err
	}
}

func (api *Router) hydrateRustSongs(ctx context.Context, ids []string) (model.MediaFiles, error) {
	if len(ids) == 0 {
		return model.MediaFiles{}, nil
	}
	values, err := api.ds.MediaFile(ctx).GetAll(model.QueryOptions{
		Filters: query.And(
			query.Eq("media_file.id", ids), query.Eq("media_file.missing", false),
		),
		ExcludeHeavyFields: true,
	})
	if err != nil {
		return nil, err
	}
	return orderRustResults(ids, values, func(value model.MediaFile) string { return value.ID }), nil
}

func (api *Router) hydrateRustAlbums(ctx context.Context, ids []string) (model.Albums, error) {
	if len(ids) == 0 {
		return model.Albums{}, nil
	}
	values, err := api.ds.Album(ctx).GetAll(model.QueryOptions{
		Filters: query.And(
			query.Eq("album.id", ids), query.Eq("album.missing", false),
		),
		ExcludeHeavyFields: true,
	})
	if err != nil {
		return nil, err
	}
	return orderRustResults(ids, values, func(value model.Album) string { return value.ID }), nil
}

func (api *Router) hydrateRustArtists(ctx context.Context, ids []string) (model.Artists, error) {
	if len(ids) == 0 {
		return model.Artists{}, nil
	}
	values, err := api.ds.Artist(ctx).GetAll(model.QueryOptions{
		Filters: query.And(
			query.Eq("artist.id", ids), query.Eq("artist.missing", false),
		),
		ExcludeHeavyFields: true,
	})
	if err != nil {
		return nil, err
	}
	return orderRustResults(ids, values, func(value model.Artist) string { return value.ID }), nil
}

func orderRustResults[T any](ids []string, values []T, getID func(T) string) []T {
	byID := make(map[string]T, len(values))
	for _, value := range values {
		byID[getID(value)] = value
	}
	ordered := make([]T, 0, len(values))
	for _, id := range ids {
		if value, ok := byID[id]; ok {
			ordered = append(ordered, value)
		}
	}
	return ordered
}

func (api *Router) Search2(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	sp, err := api.getSearchParams(r)
	if err != nil {
		return nil, err
	}

	// Get optional library IDs from musicFolderId parameter
	musicFolderIds, err := selectedMusicFolderFilterIds(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := api.searchAll(ctx, sp, musicFolderIds)

	response := newResponse()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = make([]responses.Artist, len(as))
	for i, artist := range as {
		a := responses.Artist{
			Id:             artist.ID,
			Name:           artist.Name,
			UserRating:     int32(artist.Rating),
			CoverArt:       artist.CoverArtID().String(),
			ArtistImageUrl: publicurl.ImageURL(r, artist.CoverArtID(), 600),
		}
		if artist.Starred {
			a.Starred = artist.StarredAt
		}
		searchResult2.Artist[i] = a
	}
	searchResult2.Album = albumChildren(ctx, als)
	searchResult2.Song = mediaFileChildren(ctx, mfs)
	response.SearchResult2 = searchResult2
	return response, nil
}

func (api *Router) Search3(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	sp, err := api.getSearchParams(r)
	if err != nil {
		return nil, err
	}

	// Get optional library IDs from musicFolderId parameter
	musicFolderIds, err := selectedMusicFolderFilterIds(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := api.searchAll(ctx, sp, musicFolderIds)

	response := newResponse()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = artistID3s(r, as)
	searchResult3.Album = albumID3s(ctx, als)
	searchResult3.Song = mediaFileChildren(ctx, mfs)
	response.SearchResult3 = searchResult3
	return response, nil
}

func selectedMusicFolderFilterIds(r *http.Request) ([]int, error) {
	p := req.Params(r)
	musicFolderIds, err := p.Ints("musicFolderId")
	if errors.Is(err, req.ErrMissingParam) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	accessibleLibraryIds := getUserAccessibleLibraries(r.Context()).IDs()
	for _, id := range musicFolderIds {
		if !slices.Contains(accessibleLibraryIds, id) {
			return nil, newError(responses.ErrorDataNotFound, "Library %d not found or not accessible", id)
		}
	}
	return musicFolderIds, nil
}

func albumChildren(ctx context.Context, albums model.Albums) []responses.Child {
	response := make([]responses.Child, len(albums))
	for i, album := range albums {
		response[i] = childFromAlbum(ctx, album)
	}
	return response
}

func mediaFileChildren(ctx context.Context, mediaFiles model.MediaFiles) []responses.Child {
	response := make([]responses.Child, len(mediaFiles))
	for i, mf := range mediaFiles {
		response[i] = childFromMediaFile(ctx, mf)
	}
	return response
}

func artistID3s(r *http.Request, artists model.Artists) []responses.ArtistID3 {
	response := make([]responses.ArtistID3, len(artists))
	for i, artist := range artists {
		response[i] = toArtistID3(r, artist)
	}
	return response
}

func albumID3s(ctx context.Context, albums model.Albums) []responses.AlbumID3 {
	response := make([]responses.AlbumID3, len(albums))
	for i, album := range albums {
		response[i] = buildAlbumID3(ctx, album)
	}
	return response
}
