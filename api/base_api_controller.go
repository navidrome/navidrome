package api

import (
	"encoding/xml"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
)

type BaseAPIController struct{ beego.Controller }

func (c *BaseAPIController) NewEmpty() responses.Subsonic {
	return responses.Subsonic{Status: "ok", Version: beego.AppConfig.String("apiVersion")}
}

func (c *BaseAPIController) SendError(errorCode int, message ...interface{}) {
	response := responses.Subsonic{Version: beego.AppConfig.String("apiVersion"), Status: "fail"}
	var msg string
	if len(message) == 0 {
		msg = responses.ErrorMsg(errorCode)
	} else {
		msg = fmt.Sprintf(message[0].(string), message[1:len(message)]...)
	}
	response.Error = &responses.Error{Code: errorCode, Message: msg}

	xmlBody, _ := xml.Marshal(&response)
	c.CustomAbort(200, xml.Header+string(xmlBody))
}

func (c *BaseAPIController) SendResponse(response responses.Subsonic) {
	f := c.GetString("f")
	if f == "json" {
		w := &responses.JsonWrapper{Subsonic: response}
		c.Data["json"] = &w
		c.ServeJSON()
	} else {
		c.Data["xml"] = &response
		c.ServeXML()
	}
}
