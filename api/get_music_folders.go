package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/karlkfi/inject"
	"github.com/deluan/gosonic/utils"
)

type GetMusicFoldersController struct {
	beego.Controller
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
	response := responses.NewEmpty()
	response.MusicFolders = &responses.MusicFolders{Folders: folders}
	c.Ctx.Output.Body(responses.ToXML(response))
}
