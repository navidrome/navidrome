package api

import (
	"errors"

	"github.com/astaxie/beego"
	"github.com/cloudsonic/sonic-server/api/responses"
	"github.com/cloudsonic/sonic-server/engine"
	"github.com/cloudsonic/sonic-server/utils"
)

type AlbumListController struct {
	BaseAPIController
	listGen       engine.ListGenerator
	listFunctions map[string]strategy
}

type strategy func(offset int, size int) (engine.Entries, error)

func (c *AlbumListController) Prepare() {
	utils.ResolveDependencies(&c.listGen)

	c.listFunctions = map[string]strategy{
		"random":               c.listGen.GetRandom,
		"newest":               c.listGen.GetNewest,
		"recent":               c.listGen.GetRecent,
		"frequent":             c.listGen.GetFrequent,
		"highest":              c.listGen.GetHighest,
		"alphabeticalByName":   c.listGen.GetByName,
		"alphabeticalByArtist": c.listGen.GetByArtist,
		"starred":              c.listGen.GetStarred,
	}
}

func (c *AlbumListController) getAlbumList() (engine.Entries, error) {
	typ := c.RequiredParamString("type", "Required string parameter 'type' is not present")
	listFunc, found := c.listFunctions[typ]

	if !found {
		beego.Error("albumList type", typ, "not implemented!")
		return nil, errors.New("Not implemented!")
	}

	offset := c.ParamInt("offset", 0)
	size := utils.MinInt(c.ParamInt("size", 10), 500)

	albums, err := listFunc(offset, size)
	if err != nil {
		beego.Error("Error retrieving albums:", err)
		return nil, errors.New("Internal Error")
	}

	return albums, nil
}

func (c *AlbumListController) GetAlbumList() {
	albums, err := c.getAlbumList()
	if err != nil {
		c.SendError(responses.ErrorGeneric, err.Error())
	}

	response := c.NewEmpty()
	response.AlbumList = &responses.AlbumList{Album: c.ToChildren(albums)}
	c.SendResponse(response)
}

func (c *AlbumListController) GetAlbumList2() {
	albums, err := c.getAlbumList()
	if err != nil {
		c.SendError(responses.ErrorGeneric, err.Error())
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
