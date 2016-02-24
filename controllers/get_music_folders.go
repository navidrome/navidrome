package controllers

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/controllers/responses"
)

type GetMusicFoldersController struct{ beego.Controller }

// @router /rest/getMusicFolders.view [get]
func (this *GetMusicFoldersController) Get() {
	validate(this)
	response := responses.NewError(responses.ERROR_GENERIC)
	this.Ctx.Output.Body(response)
}



