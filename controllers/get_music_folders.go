package controllers

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/controllers/responses"
	"github.com/deluan/gosonic/repositories"
)

type GetMusicFoldersController struct{ beego.Controller }

// @router /rest/getMusicFolders.view [get]
func (this *GetMusicFoldersController) Get() {
	validate(this)

	repository := new(repositories.MediaFolderRepository)
	mediaFolderList := repository.GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	musicFolders := &responses.MusicFolders{Folders: folders}
	response := responses.NewXML(musicFolders)
	this.Ctx.Output.Body(response)
}



