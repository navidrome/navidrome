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
		"random":               func(o int, s int) (engine.Entries, error) { return c.listGen.GetRandom(o, s) },
		"newest":               func(o int, s int) (engine.Entries, error) { return c.listGen.GetNewest(o, s) },
		"recent":               func(o int, s int) (engine.Entries, error) { return c.listGen.GetRecent(o, s) },
		"frequent":             func(o int, s int) (engine.Entries, error) { return c.listGen.GetFrequent(o, s) },
		"highest":              func(o int, s int) (engine.Entries, error) { return c.listGen.GetHighest(o, s) },
		"alphabeticalByName":   func(o int, s int) (engine.Entries, error) { return c.listGen.GetByName(o, s) },
		"alphabeticalByArtist": func(o int, s int) (engine.Entries, error) { return c.listGen.GetByArtist(o, s) },
		"starred":              func(o int, s int) (engine.Entries, error) { return c.listGen.GetStarred(o, s) },
	}
}

func (c *AlbumListController) GetAlbumList() {
	typ := c.RequiredParamString("type", "Required string parameter 'type' is not present")
	method, found := c.types[typ]

	if !found {
		beego.Error("albumList type", typ, "not implemented!")
		c.SendError(responses.ErrorGeneric, "Not implemented!")
	}

	offset := c.ParamInt("offset", 0)
	size := utils.MinInt(c.ParamInt("size", 10), 500)

	albums, err := method(offset, size)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.AlbumList = &responses.AlbumList{Album: c.ToChildren(albums)}
	c.SendResponse(response)
}

func (c *AlbumListController) GetAlbumList2() {
	typ := c.RequiredParamString("type", "Required string parameter 'type' is not present")
	method, found := c.types[typ]

	if !found {
		beego.Error("albumList2 type", typ, "not implemented!")
		c.SendError(responses.ErrorGeneric, "Not implemented!")
	}

	offset := c.ParamInt("offset", 0)
	size := utils.MinInt(c.ParamInt("size", 10), 500)

	albums, err := method(offset, size)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.AlbumList2 = &responses.AlbumList{Album: c.ToAlbums(albums)}
	c.SendResponse(response)
}

func (c *AlbumListController) GetStarred() {
	albums, mediaFiles, err := c.listGen.GetAllStarred()
	if err != nil {
		beego.Error("Error retrieving starred media:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.Starred = &responses.Starred{}
	response.Starred.Album = c.ToChildren(albums)
	response.Starred.Song = c.ToChildren(mediaFiles)

	c.SendResponse(response)
}

func (c *AlbumListController) GetStarred2() {
	albums, mediaFiles, err := c.listGen.GetAllStarred()
	if err != nil {
		beego.Error("Error retrieving starred media:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.Starred2 = &responses.Starred{}
	response.Starred2.Album = c.ToAlbums(albums)
	response.Starred2.Song = c.ToChildren(mediaFiles)

	c.SendResponse(response)
}

func (c *AlbumListController) GetNowPlaying() {
	npInfos, err := c.listGen.GetNowPlaying()
	if err != nil {
		beego.Error("Error retrieving now playing list:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
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

func (c *AlbumListController) GetRandomSongs() {
	size := utils.MinInt(c.ParamInt("size", 10), 500)

	songs, err := c.listGen.GetRandomSongs(size)
	if err != nil {
		beego.Error("Error retrieving random songs:", err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.RandomSongs = &responses.Songs{}
	response.RandomSongs.Songs = make([]responses.Child, len(songs))
	for i, entry := range songs {
		response.RandomSongs.Songs[i] = c.ToChild(entry)
	}
	c.SendResponse(response)
}
