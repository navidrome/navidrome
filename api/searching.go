package api

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type SearchingController struct {
	BaseAPIController
	search       engine.Search
	query        string
	artistCount  int
	artistOffset int
	albumCount   int
	albumOffset  int
	songCount    int
	songOffset   int
}

func (c *SearchingController) Prepare() {
	utils.ResolveDependencies(&c.search)
}

func (c *SearchingController) getParams() {
	c.query = c.RequiredParamString("query", "Parameter query required")
	c.artistCount = c.ParamInt("artistCount", 20)
	c.artistOffset = c.ParamInt("artistOffset", 0)
	c.albumCount = c.ParamInt("albumCount", 20)
	c.albumOffset = c.ParamInt("albumOffset", 0)
	c.songCount = c.ParamInt("songCount", 20)
	c.songOffset = c.ParamInt("songOffset", 0)
}

func (c *SearchingController) searchAll() (engine.Entries, engine.Entries, engine.Entries) {
	as, err := c.search.SearchArtist(c.query, c.artistOffset, c.artistCount)
	if err != nil {
		beego.Error("Error searching for Artists:", err)
	}
	als, err := c.search.SearchAlbum(c.query, c.albumOffset, c.albumCount)
	if err != nil {
		beego.Error("Error searching for Albums:", err)
	}
	mfs, err := c.search.SearchSong(c.query, c.songOffset, c.songCount)
	if err != nil {
		beego.Error("Error searching for MediaFiles:", err)
	}

	beego.Debug(fmt.Sprintf("Searching for [%s] resulted in %d songs, %d albums and %d artists", c.query, len(mfs), len(als), len(as)))
	return mfs, als, as
}

func (c *SearchingController) Search2() {
	c.getParams()
	mfs, als, as := c.searchAll()

	response := c.NewEmpty()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = make([]responses.Artist, len(as))
	for i, e := range as {
		searchResult2.Artist[i] = responses.Artist{Id: e.Id, Name: e.Title}
	}
	searchResult2.Album = c.ToChildren(als)
	searchResult2.Song = c.ToChildren(mfs)
	response.SearchResult2 = searchResult2
	c.SendResponse(response)
}

func (c *SearchingController) Search3() {
	c.getParams()
	mfs, als, as := c.searchAll()

	response := c.NewEmpty()
	searchResult3 := &responses.SearchResult3{}
	searchResult3.Artist = make([]responses.ArtistID3, len(as))
	for i, e := range as {
		searchResult3.Artist[i] = responses.ArtistID3{
			Id:         e.Id,
			Name:       e.Title,
			CoverArt:   e.CoverArt,
			AlbumCount: e.AlbumCount,
		}
	}
	searchResult3.Album = c.ToAlbums(als)
	searchResult3.Song = c.ToChildren(mfs)
	response.SearchResult3 = searchResult3
	c.SendResponse(response)
}
