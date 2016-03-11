package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type SearchingController struct {
	BaseAPIController
	search engine.Search
}

func (c *SearchingController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.search)
}

func (c *SearchingController) Search2() {
	query := c.RequiredParamString("query", "Parameter query required")
	artistCount := c.ParamInt("artistCount", 20)
	artistOffset := c.ParamInt("artistOffset", 0)
	//albumCount := c.ParamInt("albumCount", 20)
	//albumOffset := c.ParamInt("albumOffset", 0)
	//songCount := c.ParamInt("songCount", 20)
	//songOffset := c.ParamInt("songOffset", 0)

	as, err := c.search.SearchArtist(query, artistOffset, artistCount)
	if err != nil {
		beego.Error("Error searching for Artists:", err)
	}
	//als, err := c.search.SearchAlbum(query, albumOffset, albumCount)
	//if err != nil {
	//	beego.Error("Error searching for Albums:", err)
	//}
	//mfs, err := c.search.SearchSong(query, songOffset, songCount)
	//if err != nil {
	//	beego.Error("Error searching for MediaFiles:", err)
	//}

	response := c.NewEmpty()
	searchResult2 := &responses.SearchResult2{}
	searchResult2.Artist = make([]responses.Artist, len(*as))
	for i, a := range *as {
		searchResult2.Artist[i] = responses.Artist{Id: a.Id, Name: a.Title}
	}
	//searchResult2.Album = make([]responses.Child, len(*as))
	//for i, a := range *as {
	//	searchResult2.Album[i] = responses.Child{Id: a.Id, Name: a.Name}
	//}
	//searchResult2.Artist = make([]responses.Artist, len(*as))
	//for i, a := range *as {
	//	searchResult2.Artist[i] = responses.Artist{Id: a.Id, Name: a.Name}
	//}
	response.SearchResult2 = searchResult2
	c.SendResponse(response)
}
