package subsonic

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/kennygrant/sanitize"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server/subsonic/responses"
	"github.com/navidrome/navidrome/utils"
)

type SearchingController struct {
	ds model.DataStore
}

type searchParams struct {
	query        string
	artistCount  int
	artistOffset int
	albumCount   int
	albumOffset  int
	songCount    int
	songOffset   int
}

func NewSearchingController(ds model.DataStore) *SearchingController {
	return &SearchingController{ds: ds}
}

func (c *SearchingController) getParams(r *http.Request) (*searchParams, error) {
	var err error
	sp := &searchParams{}
	sp.query, err = requiredParamString(r, "query")
	if err != nil {
		return nil, err
	}
	sp.artistCount = utils.ParamInt(r, "artistCount", 20)
	sp.artistOffset = utils.ParamInt(r, "artistOffset", 0)
	sp.albumCount = utils.ParamInt(r, "albumCount", 20)
	sp.albumOffset = utils.ParamInt(r, "albumOffset", 0)
	sp.songCount = utils.ParamInt(r, "songCount", 20)
	sp.songOffset = utils.ParamInt(r, "songOffset", 0)
	return sp, nil
}

type searchFunc[T any] func(q string, offset int, size int) (T, error)

func doSearch[T any](ctx context.Context, wg *sync.WaitGroup, s searchFunc[T], q string, offset, size int) T {
	defer wg.Done()
	var res T
	if size == 0 {
		return res
	}
	done := make(chan struct{})
	go func() {
		typ := reflect.TypeOf(res).String()
		var err error
		start := time.Now()
		res, err = s(q, offset, size)
		if err != nil {
			log.Error(ctx, "Error searching "+typ, err)
		} else {
			log.Trace(ctx, "Search for "+typ+" completed", "elapsedTime", time.Since(start))
		}
		done <- struct{}{}
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
	return res
}

func (c *SearchingController) searchAll(r *http.Request, sp *searchParams) (mediaFiles model.MediaFiles, albums model.Albums, artists model.Artists) {
	start := time.Now()
	q := sanitize.Accents(strings.ToLower(strings.TrimSuffix(sp.query, "*")))
	ctx := r.Context()
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() { mediaFiles = doSearch(ctx, wg, c.ds.MediaFile(ctx).Search, q, sp.songOffset, sp.songCount) }()
	go func() { albums = doSearch(ctx, wg, c.ds.Album(ctx).Search, q, sp.albumOffset, sp.albumCount) }()
	go func() { artists = doSearch(ctx, wg, c.ds.Artist(ctx).Search, q, sp.artistOffset, sp.artistCount) }()
	wg.Wait()

	if ctx.Err() == nil {
		log.Debug(ctx, fmt.Sprintf("Search resulted in %d songs, %d albums and %d artists",
			len(mediaFiles), len(albums), len(artists)), "query", sp.query, "elapsedTime", time.Since(start))
	} else {
		log.Warn(ctx, "Search was interrupted", ctx.Err(), "query", sp.query, "elapsedTime", time.Since(start))
	}
	return mediaFiles, albums, artists
}

func (c *SearchingController) Search2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	sp, err := c.getParams(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := c.searchAll(r, sp)

	response := newResponse()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = make([]responses.Artist, len(as))
	for i, artist := range as {
		searchResult2.Artist[i] = responses.Artist{
			Id:         artist.ID,
			Name:       artist.Name,
			AlbumCount: artist.AlbumCount,
			UserRating: artist.Rating,
		}
		if artist.Starred {
			searchResult2.Artist[i].Starred = &artist.StarredAt
		}
	}
	searchResult2.Album = childrenFromAlbums(r.Context(), als)
	searchResult2.Song = childrenFromMediaFiles(r.Context(), mfs)
	response.SearchResult2 = searchResult2
	return response, nil
}

func (c *SearchingController) Search3(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	ctx := r.Context()
	sp, err := c.getParams(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := c.searchAll(r, sp)

	response := newResponse()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = make([]responses.ArtistID3, len(as))
	for i, artist := range as {
		searchResult3.Artist[i] = toArtistID3(ctx, artist)
	}
	searchResult3.Album = childrenFromAlbums(ctx, als)
	searchResult3.Song = childrenFromMediaFiles(ctx, mfs)
	response.SearchResult3 = searchResult3
	return response, nil
}
