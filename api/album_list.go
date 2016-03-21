package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type AlbumListController struct {
	BaseAPIController
	listGen engine.ListGenerator
	types   map[string]strategy
}

type strategy func(offset int, size int) (engine.Entries, error)

func (c *AlbumListController) Prepare() {
	utils.ResolveDependencies(&c.listGen)

	c.types = map[string]strategy{
		"random":   func(o int, s int) (engine.Entries, error) { return c.listGen.GetRandom(o, s) },
		"newest":   func(o int, s int) (engine.Entries, error) { return c.listGen.GetNewest(o, s) },
		"recent":   func(o int, s int) (engine.Entries, error) { return c.listGen.GetRecent(o, s) },
		"frequent": func(o int, s int) (engine.Entries, error) { return c.listGen.GetFrequent(o, s) },
		"highest":  func(o int, s int) (engine.Entries, error) { return c.listGen.GetHighest(o, s) },
	}
}

func (c *AlbumListController) GetAlbumList() {
	typ := c.RequiredParamString("type", "Required string parameter 'type' is not present")
	method, found := c.types[typ]

	if !found {
		beego.Error("albumList type", typ, "not implemented!")
		c.SendError(responses.ERROR_GENERIC, "Not implemented!")
	}

	offset := c.ParamInt("offset", 0)
	size := utils.MinInt(c.ParamInt("size", 0), 500)

	albums, err := method(offset, size)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response := c.NewEmpty()
	response.AlbumList = &responses.AlbumList{Album: c.ToChildren(albums)}
	c.SendResponse(response)
}

func (c *AlbumListController) GetStarred() {
	albums, err := c.listGen.GetStarred()
	if err != nil {
		beego.Error("Error retrieving starred albums:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response := c.NewEmpty()
	response.Starred = &responses.Starred{}
	response.Starred.Album = c.ToChildren(albums)

	c.SendResponse(response)
}

func (c *AlbumListController) GetNowPlaying() {
	npInfos, err := c.listGen.GetNowPlaying()
	if err != nil {
		beego.Error("Error retrieving now playing list:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response := c.NewEmpty()
	response.NowPlaying = &responses.NowPlaying{}
	response.NowPlaying.Entry = make([]responses.NowPlayingEntry, len(npInfos))
	for i, entry := range npInfos {
		response.NowPlaying.Entry[i].Child = c.ToChild(entry)
		response.NowPlaying.Entry[i].UserName = entry.UserName
		response.NowPlaying.Entry[i].MinutesAgo = entry.MinutesAgo
		response.NowPlaying.Entry[i].PlayerId = entry.PlayerId
		response.NowPlaying.Entry[i].PlayerName = entry.PlayerName
	}
	c.SendResponse(response)
}
