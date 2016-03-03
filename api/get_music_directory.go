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

	if found {
		_, err := c.artistRepo.Get(id)
		if err != nil {
			beego.Error("Error reading Artist from DB", err)
			c.SendError(responses.ERROR_GENERIC, "Internal Error")
		}
	}

	response := c.NewEmpty()
	response.Directory = &responses.Directory{}
	c.SendResponse(response)
}
