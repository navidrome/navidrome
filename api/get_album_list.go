package api

import (
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetAlbumListController struct {
	BaseAPIController
	listGen engine.ListGenerator
	types   map[string]strategy
}

type strategy func(offset int, size int) (*domain.Albums, error)

func (c *GetAlbumListController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.listGen)

	c.types = map[string]strategy{
		"random":   func(o int, s int) (*domain.Albums, error) { return c.listGen.GetRandom(o, s) },
		"newest":   func(o int, s int) (*domain.Albums, error) { return c.listGen.GetNewest(o, s) },
		"recent":   func(o int, s int) (*domain.Albums, error) { return c.listGen.GetRecent(o, s) },
		"frequent": func(o int, s int) (*domain.Albums, error) { return c.listGen.GetFrequent(o, s) },
		"highest":  func(o int, s int) (*domain.Albums, error) { return c.listGen.GetHighest(o, s) },
	}
}

func (c *GetAlbumListController) Get() {
	typ := c.RequiredParamString("type", "Required string parameter 'type' is not present")
	method, found := c.types[typ]

	if !found {
		beego.Error("getAlbumList type", typ, "not implemented!")
		c.SendError(responses.ERROR_GENERIC, "Not implemented!")
	}

	offset := c.ParamInt("offset")
	size := utils.MinInt(c.ParamInt("size"), 500)

	albums, err := method(offset, size)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	albumList := make([]responses.Child, len(*albums))

	for i, al := range *albums {
		albumList[i].Id = al.Id
		albumList[i].Title = al.Name
		albumList[i].Parent = al.ArtistId
		albumList[i].IsDir = true
		albumList[i].Album = al.Name
		albumList[i].Year = al.Year
		albumList[i].Artist = al.Artist
		albumList[i].Genre = al.Genre
		albumList[i].CoverArt = al.CoverArtId
		if al.Starred {
			t := time.Now()
			albumList[i].Starred = &t
		}
	}

	response := c.NewEmpty()
	response.AlbumList = &responses.AlbumList{Album: albumList}
	c.SendResponse(response)
}
