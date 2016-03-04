package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"time"
)

type GetAlbumListController struct {
	BaseAPIController
	albumRepo domain.AlbumRepository
	types     map[string]domain.QueryOptions
}

func (c *GetAlbumListController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.albumRepo)

	c.types = map[string]domain.QueryOptions{
		"newest":   domain.QueryOptions{SortBy: "CreatedAt", Desc: true, Alpha: true},
		"recent":   domain.QueryOptions{SortBy: "PlayDate", Desc: true, Alpha: true},
		"frequent": domain.QueryOptions{SortBy: "PlayCount", Desc: true},
		"highest":  domain.QueryOptions{SortBy: "Rating", Desc: true},
	}
}

func (c *GetAlbumListController) Get() {
	typ := c.GetParameter("type", "Required string parameter 'type' is not present")
	qo, found := c.types[typ]

	if !found {
		beego.Error("getAlbumList type", typ, "not implemented!")
		c.SendError(responses.ERROR_GENERIC, "Not implemented yet!")
	}

	qo.Size = 10
	c.Ctx.Input.Bind(&qo.Size, "size")
	c.Ctx.Input.Bind(&qo.Offset, "offset")

	albums, err := c.albumRepo.GetAll(qo)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	albumList := make([]responses.Child, len(albums))

	for i, al := range albums {
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
