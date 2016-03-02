package api

import (
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetMusicFoldersController struct {
	BaseAPIController
	repo domain.MediaFolderRepository
}

func (c *GetMusicFoldersController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.repo)
}

func (c *GetMusicFoldersController) Get() {
	mediaFolderList, _ := c.repo.GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	response := c.NewEmpty()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	c.SendResponse(response)
}
