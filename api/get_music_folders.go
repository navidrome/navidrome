package api

import (
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetMusicFoldersController struct {
	BaseAPIController
	browser engine.Browser
}

func (c *GetMusicFoldersController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.browser)
}

func (c *GetMusicFoldersController) Get() {
	mediaFolderList, _ := c.browser.MediaFolders()
	folders := make([]responses.MusicFolder, len(*mediaFolderList))
	for i, f := range *mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	response := c.NewEmpty()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	c.SendResponse(response)
}
