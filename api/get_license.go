package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type GetLicenseController struct{ beego.Controller }

func (this *GetLicenseController) Get() {
	response := responses.NewXML(&responses.License{Valid: true})
	this.Ctx.Output.Body(response)
}



