package controllers

import (
	"github.com/astaxie/beego"
	"encoding/xml"
	"github.com/deluan/gosonic/responses"
)

type GetLicenseController struct{ beego.Controller }

// @router /rest/getLicense.view [get]
func (this *GetLicenseController) Get() {
	response := responses.NewGetLicense(true)
	xmlBody, _ := xml.Marshal(response)
	this.Ctx.Output.Body([]byte(xml.Header + string(xmlBody)))
}



