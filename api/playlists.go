package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type PlaylistsController struct {
	BaseAPIController
	pls engine.Playlists
}

func (c *PlaylistsController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.pls)
}

func (c *PlaylistsController) GetAll() {
	allPls, err := c.pls.GetAll()
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}
	playlists := make([]responses.Playlist, len(*allPls))
	for i, f := range *allPls {
		playlists[i].Id = f.Id
		playlists[i].Name = f.Name
	}
	response := c.NewEmpty()
	response.Playlists = &responses.Playlists{Playlist: playlists}
	c.SendResponse(response)
}
