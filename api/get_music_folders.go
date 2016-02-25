package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/repositories"
)

type GetMusicFoldersController struct{ beego.Controller }

func (this *GetMusicFoldersController) Get() {
	repository := new(repositories.MediaFolderRepository)
	mediaFolderList, _ := repository.GetAll()
	folders := make([]responses.MusicFolder, len(mediaFolderList))
	for i, f := range mediaFolderList {
		folders[i].Id = f.Id
		folders[i].Name = f.Name
	}
	musicFolders := &responses.MusicFolders{Folders: folders}
	response := responses.NewXML(musicFolders)
	this.Ctx.Output.Body(response)
}



