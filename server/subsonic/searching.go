package subsonic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
	"github.com/kennygrant/sanitize"
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

func (c *SearchingController) searchAll(r *http.Request, sp *searchParams) (model.MediaFiles, model.Albums, model.Artists) {
	q := sanitize.Accents(strings.ToLower(strings.TrimSuffix(sp.query, "*")))
	ctx := r.Context()

	artists, err := c.ds.Artist(ctx).Search(q, sp.artistOffset, sp.artistCount)
	if err != nil {
		log.Error(ctx, "Error searching for Artists", err)
	}
	albums, err := c.ds.Album(ctx).Search(q, sp.albumOffset, sp.albumCount)
	if err != nil {
		log.Error(ctx, "Error searching for Albums", err)
	}
	mediaFiles, err := c.ds.MediaFile(ctx).Search(q, sp.songOffset, sp.songCount)
	if err != nil {
		log.Error(ctx, "Error searching for MediaFiles", err)
	}

	log.Debug(ctx, fmt.Sprintf("Search resulted in %d songs, %d albums and %d artists",
		len(mediaFiles), len(albums), len(artists)), "query", sp.query)
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
	sp, err := c.getParams(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := c.searchAll(r, sp)

	response := newResponse()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = make([]responses.ArtistID3, len(as))
	for i, artist := range as {
		searchResult3.Artist[i] = responses.ArtistID3{
			Id:         artist.ID,
			Name:       artist.Name,
			AlbumCount: artist.AlbumCount,
		}
		if artist.Starred {
			searchResult3.Artist[i].Starred = &artist.StarredAt
		}
	}
	searchResult3.Album = childrenFromAlbums(r.Context(), als)
	searchResult3.Song = childrenFromMediaFiles(r.Context(), mfs)
	response.SearchResult3 = searchResult3
	return response, nil
}
