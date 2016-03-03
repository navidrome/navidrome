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
	albumRepo domain.AlbumRepository
}

func (c *GetMusicDirectoryController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.artistRepo)
	inject.ExtractAssignable(utils.Graph, &c.albumRepo)
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
		a, albums := c.retrieveArtist(id)

		dir.Id = a.Id
		dir.Name = a.Name
		dir.Child = make([]responses.Child, len(albums))
		for i, al := range albums {
			dir.Child[i].Id = al.Id
			dir.Child[i].Title = al.Name
			dir.Child[i].IsDir = true
			dir.Child[i].Album = al.Name
			dir.Child[i].Year = al.Year
			dir.Child[i].Artist = a.Name
		}
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

	if as, err = c.albumRepo.FindByArtist(id); err != nil {
		beego.Error("Error reading Album from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return
}