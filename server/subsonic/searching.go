package subsonic

import (
	"fmt"
	"net/http"

	"github.com/deluan/navidrome/engine"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/server/subsonic/responses"
	"github.com/deluan/navidrome/utils"
)

type SearchingController struct {
	search engine.Search
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

func NewSearchingController(search engine.Search) *SearchingController {
	return &SearchingController{search: search}
}

func (c *SearchingController) getParams(r *http.Request) (*searchParams, error) {
	var err error
	sp := &searchParams{}
	sp.query, err = RequiredParamString(r, "query", "Parameter query required")
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

func (c *SearchingController) searchAll(r *http.Request, sp *searchParams) (engine.Entries, engine.Entries, engine.Entries) {
	as, err := c.search.SearchArtist(r.Context(), sp.query, sp.artistOffset, sp.artistCount)
	if err != nil {
		log.Error(r, "Error searching for Artists", err)
	}
	als, err := c.search.SearchAlbum(r.Context(), sp.query, sp.albumOffset, sp.albumCount)
	if err != nil {
		log.Error(r, "Error searching for Albums", err)
	}
	mfs, err := c.search.SearchSong(r.Context(), sp.query, sp.songOffset, sp.songCount)
	if err != nil {
		log.Error(r, "Error searching for MediaFiles", err)
	}

	log.Debug(r, fmt.Sprintf("Search resulted in %d songs, %d albums and %d artists", len(mfs), len(als), len(as)), "query", sp.query)
	return mfs, als, as
}

func (c *SearchingController) Search2(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	sp, err := c.getParams(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := c.searchAll(r, sp)

	response := NewResponse()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = ToArtists(as)
	searchResult2.Album = ToChildren(als)
	searchResult2.Song = ToChildren(mfs)
	response.SearchResult2 = searchResult2
	return response, nil
}

func (c *SearchingController) Search3(w http.ResponseWriter, r *http.Request) (*responses.Subsonic, error) {
	sp, err := c.getParams(r)
	if err != nil {
		return nil, err
	}
	mfs, als, as := c.searchAll(r, sp)

	response := NewResponse()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = make([]responses.ArtistID3, len(as))
	for i, e := range as {
		searchResult3.Artist[i] = responses.ArtistID3{
			Id:         e.Id,
			Name:       e.Title,
			CoverArt:   e.CoverArt,
			AlbumCount: e.AlbumCount,
		}
		if !e.Starred.IsZero() {
			searchResult3.Artist[i].Starred = &e.Starred
		}
	}
	searchResult3.Album = ToAlbums(als)
	searchResult3.Song = ToChildren(mfs)
	response.SearchResult3 = searchResult3
	return response, nil
}
