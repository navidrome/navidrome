package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type GetLicenseController struct{ beego.Controller }

func (c *GetLicenseController) Get() {
	response := responses.NewXML(&responses.License{Valid: true})
	c.Ctx.Output.Body(response)
}



