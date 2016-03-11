package api

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type BaseAPIController struct{ beego.Controller }

func (c *BaseAPIController) NewEmpty() responses.Subsonic {
	return responses.Subsonic{Status: "ok", Version: beego.AppConfig.String("apiVersion")}
}

func (c *BaseAPIController) RequiredParamString(param string, msg string) string {
	p := c.Input().Get(param)
	if p == "" {
		c.SendError(responses.ERROR_MISSING_PARAMETER, msg)
	}
	return p
}

func (c *BaseAPIController) ParamString(param string) string {
	return c.Input().Get(param)
}

func (c *BaseAPIController) ParamTime(param string) time.Time {
	var value int64
	c.Ctx.Input.Bind(&value, param)
	return utils.ToTime(value)
}

func (c *BaseAPIController) ParamInt(param string, def int) int {
	value := def
	c.Ctx.Input.Bind(&value, param)
	return value
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
	switch f {
	case "json":
		w := &responses.JsonWrapper{Subsonic: response}
		c.Data["json"] = &w
		c.ServeJSON()
	case "jsonp":
		w := &responses.JsonWrapper{Subsonic: response}
		c.Data["jsonp"] = &w
		c.ServeJSONP()
	default:
		c.Data["xml"] = &response
		c.ServeXML()
	}
}

func (c *BaseAPIController) ToChild(entry engine.Entry) responses.Child {
	n := responses.Child{}
	n.Id = entry.Id
	n.Title = entry.Title
	n.IsDir = entry.IsDir
	n.Parent = entry.Parent
	n.Album = entry.Album
	n.Year = entry.Year
	n.Artist = entry.Artist
	n.Genre = entry.Genre
	n.CoverArt = entry.CoverArt
	n.Track = entry.Track
	n.Duration = entry.Duration
	n.Size = entry.Size
	n.Suffix = entry.Suffix
	n.BitRate = entry.BitRate
	n.ContentType = entry.ContentType
	if !entry.Starred.IsZero() {
		n.Starred = &entry.Starred
	}
	return n
}
