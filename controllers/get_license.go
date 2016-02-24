package controllers

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/responses"
)

type GetLicenseController struct{ beego.Controller }

// @router /rest/getLicense.view [get]
func (this *GetLicenseController) Get() {
	response := responses.NewXML(&responses.License{Valid: true})
	this.Ctx.Output.Body(response)
}



