package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/sanitize"
	"github.com/navidrome/navidrome/core/publicurl"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils/req"
	"github.com/navidrome/navidrome/utils/slice"
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

func (api *Router) getSearchParams(r *http.Request) (*searchParams, error) {
	p := req.Params(r)
	sp := &searchParams{}
	sp.query = p.StringOr("query", `""`)
	sp.artistCount = p.IntOr("artistCount", 20)
	sp.artistOffset = p.IntOr("artistOffset", 0)
	sp.albumCount = p.IntOr("albumCount", 20)
	sp.albumOffset = p.IntOr("albumOffset", 0)
	sp.songCount = p.IntOr("songCount", 20)
	sp.songOffset = p.IntOr("songOffset", 0)
	return sp, nil
}

type searchFunc[T any] func(q string, options ...model.QueryOptions) (T, error)

func callSearch[T any](ctx context.Context, s searchFunc[T], q string, options model.QueryOptions, result *T) func() error {
	return func() error {
		if options.Max == 0 {
			return nil
		}
		typ := strings.TrimPrefix(reflect.TypeOf(*result).String(), "model.")
		var err error
		start := time.Now()
		*result, err = s(q, options)
		if err != nil {
			log.Error(ctx, "Error searching "+typ, "query", q, "elapsed", time.Since(start), err)
		} else {
			log.Trace(ctx, "Search for "+typ+" completed", "query", q, "elapsed", time.Since(start))
		}
		return nil
	}
}

func (api *Router) searchAll(ctx context.Context, sp *searchParams, musicFolderIds []int) (mediaFiles model.MediaFiles, albums model.Albums, artists model.Artists) {
	start := time.Now()
	q := sanitize.Accents(strings.ToLower(strings.TrimSuffix(sp.query, "*")))

	// Build options with offset/size/filters packed in
	songOpts := model.QueryOptions{Max: sp.songCount, Offset: sp.songOffset}
	albumOpts := model.QueryOptions{Max: sp.albumCount, Offset: sp.albumOffset}
	artistOpts := model.QueryOptions{Max: sp.artistCount, Offset: sp.artistOffset}

	if len(musicFolderIds) > 0 {
		songOpts.Filters = Eq{"library_id": musicFolderIds}
		albumOpts.Filters = Eq{"library_id": musicFolderIds}
		artistOpts.Filters = Eq{"library_artist.library_id": musicFolderIds}
	}

	// Run searches in parallel
	g, ctx := errgroup.WithContext(ctx)
	g.Go(callSearch(ctx, api.ds.MediaFile(ctx).Search, q, songOpts, &mediaFiles))
	g.Go(callSearch(ctx, api.ds.Album(ctx).Search, q, albumOpts, &albums))
	g.Go(callSearch(ctx, api.ds.Artist(ctx).Search, q, artistOpts, &artists))
	err := g.Wait()
	if err == nil {
		log.Debug(ctx, fmt.Sprintf("Search resulted in %d songs, %d albums and %d artists",
			len(mediaFiles), len(albums), len(artists)), "query", sp.query, "elapsedTime", time.Since(start))
	} else {
		log.Warn(ctx, "Search was interrupted", "query", sp.query, "elapsedTime", time.Since(start), err)
	}
	return mediaFiles, albums, artists
}

func (api *Router) Search2(r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	sp, err := api.getSearchParams(r)
	if err != nil {
		return nil, err
	}

	// Get optional library IDs from musicFolderId parameter
	musicFolderIds, err := selectedMusicFolderIds(r, false)
	if err != nil {
		return nil, err
	}
	mfs, als, as := api.searchAll(ctx, sp, musicFolderIds)

	response := newResponse()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = slice.Map(as, func(artist model.Artist) responses.Artist {
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
		return a
	})
	searchResult2.Album = slice.MapWithArg(als, ctx, childFromAlbum)
	searchResult2.Song = slice.MapWithArg(mfs, ctx, childFromMediaFile)
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
	musicFolderIds, err := selectedMusicFolderIds(r, false)
	if err != nil {
		return nil, err
	}
	mfs, als, as := api.searchAll(ctx, sp, musicFolderIds)

	response := newResponse()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = slice.MapWithArg(as, r, toArtistID3)
	searchResult3.Album = slice.MapWithArg(als, ctx, buildAlbumID3)
	searchResult3.Song = slice.MapWithArg(mfs, ctx, childFromMediaFile)
	response.SearchResult3 = searchResult3
	return response, nil
}
