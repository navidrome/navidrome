package controllers

import (
	"github.com/astaxie/beego"
	"encoding/xml"
)

type PingResponse struct {
	XMLName xml.Name `xml:"http://subsonic.org/restapi subsonic-response"`
	Status  string   `xml:"status,attr"`
	Version string   `xml:"version,attr"`
}

type PingController struct{ beego.Controller }

// @router /rest/ping.view [get]
func (this *PingController) Get() {
	response := &PingResponse{Status:"ok", Version: beego.AppConfig.String("apiversion")}
	xmlBody, _ := xml.Marshal(response)
	this.Ctx.Output.Body([]byte(xml.Header + string(xmlBody)))
}



