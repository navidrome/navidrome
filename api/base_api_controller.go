package api

import (
	"encoding/xml"
	"fmt"
	"time"

	"strconv"

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
		c.SendError(responses.ErrorMissingParameter, msg)
	}
	return p
}

func (c *BaseAPIController) RequiredParamStrings(param string, msg string) []string {
	ps := c.Input()[param]
	if len(ps) == 0 {
		c.SendError(responses.ErrorMissingParameter, msg)
	}
	return ps
}

func (c *BaseAPIController) ParamString(param string) string {
	return c.Input().Get(param)
}

func (c *BaseAPIController) ParamStrings(param string) []string {
	return c.Input()[param]
}

func (c *BaseAPIController) ParamTime(param string, def time.Time) time.Time {
	var value int64
	if c.Input().Get(param) == "" {
		return def
	}
	c.Ctx.Input.Bind(&value, param)
	return utils.ToTime(value)
}

func (c *BaseAPIController) ParamTimes(param string) []time.Time {
	pStr := c.Input()[param]
	times := make([]time.Time, len(pStr))
	for i, t := range pStr {
		ti, err := strconv.ParseInt(t, 10, 64)
		if err == nil {
			times[i] = utils.ToTime(ti)
		}
	}
	return times
}

func (c *BaseAPIController) RequiredParamInt(param string, msg string) int {
	p := c.Input().Get(param)
	if p == "" {
		c.SendError(responses.ErrorMissingParameter, msg)
	}
	return c.ParamInt(param, 0)
}

func (c *BaseAPIController) ParamInt(param string, def int) int {
	value := def
	c.Ctx.Input.Bind(&value, param)
	return value
}

func (c *BaseAPIController) ParamInts(param string) []int {
	pStr := c.Input()[param]
	ints := make([]int, 0, len(pStr))
	for _, s := range pStr {
		i, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			ints = append(ints, int(i))
		}
	}
	return ints
}

func (c *BaseAPIController) ParamBool(param string, def bool) bool {
	value := def
	if c.Input().Get(param) == "" {
		return def
	}
	c.Ctx.Input.Bind(&value, param)
	return value
}

func (c *BaseAPIController) SendError(errorCode int, message ...interface{}) {
	response := responses.Subsonic{Version: beego.AppConfig.String("apiVersion"), Status: "fail"}
	var msg string
	if len(message) == 0 {
		msg = responses.ErrorMsg(errorCode)
	} else {
		msg = fmt.Sprintf(message[0].(string), message[1:]...)
	}
	response.Error = &responses.Error{Code: errorCode, Message: msg}

	xmlBody, _ := xml.Marshal(&response)
	c.CustomAbort(200, xml.Header+string(xmlBody))
}

func (c *BaseAPIController) SendEmptyResponse() {
	c.SendResponse(c.NewEmpty())
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

func (c *BaseAPIController) ToChildren(entries engine.Entries) []responses.Child {
	children := make([]responses.Child, len(entries))
	for i, entry := range entries {
		children[i] = c.ToChild(entry)
	}
	return children
}

func (c *BaseAPIController) ToChild(entry engine.Entry) responses.Child {
	child := responses.Child{}
	child.Id = entry.Id
	child.Title = entry.Title
	child.IsDir = entry.IsDir
	child.Parent = entry.Parent
	child.Album = entry.Album
	child.Year = entry.Year
	child.Artist = entry.Artist
	child.Genre = entry.Genre
	child.CoverArt = entry.CoverArt
	child.Track = entry.Track
	child.Duration = entry.Duration
	child.Size = entry.Size
	child.Suffix = entry.Suffix
	child.BitRate = entry.BitRate
	child.ContentType = entry.ContentType
	if !entry.Starred.IsZero() {
		child.Starred = &entry.Starred
	}
	child.Path = entry.Path
	child.PlayCount = entry.PlayCount
	child.DiscNumber = entry.DiscNumber
	if !entry.Created.IsZero() {
		child.Created = &entry.Created
	}
	child.AlbumId = entry.AlbumId
	child.ArtistId = entry.ArtistId
	child.Type = entry.Type
	child.UserRating = entry.UserRating
	return child
}
