package api

import (
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"github.com/astaxie/beego"
)

type GetMusicDirectoryController struct {
	BaseAPIController
	artistRepo domain.ArtistRepository
}

func (c *GetMusicDirectoryController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.artistRepo)
}

func (c *GetMusicDirectoryController) Get() {
	id := c.Input().Get("id")

	if id == "" {
		c.SendError(responses.ERROR_MISSING_PARAMETER, "id parameter required")
	}

	found, err := c.artistRepo.Exists(id)
	if err != nil {
		beego.Error("Error searching for Artist:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	dir := &responses.Directory{}
	if found {
		a, _:= c.retrieveArtist(id)

		dir.Id = a.Id
		dir.Name = a.Name
	} else {
		beego.Info("Artist", id, "not found")
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	}

	response := c.NewEmpty()
	response.Directory = dir
	c.SendResponse(response)
}

func (c *GetMusicDirectoryController) retrieveArtist(id string) (a *domain.Artist, as[]domain.Album) {
	var err error

	if a, err = c.artistRepo.Get(id); err != nil {
		beego.Error("Error reading Artist from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	as = make([]domain.Album, 0)
	return
}