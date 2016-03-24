package api

import (
	"fmt"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
)

type PlaylistsController struct {
	BaseAPIController
	pls engine.Playlists
}

func (c *PlaylistsController) Prepare() {
	utils.ResolveDependencies(&c.pls)
}

func (c *PlaylistsController) GetPlaylists() {
	allPls, err := c.pls.GetAll()
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal error")
	}
	playlists := make([]responses.Playlist, len(allPls))
	for i, p := range allPls {
		playlists[i].Id = p.Id
		playlists[i].Name = p.Name
		playlists[i].Comment = p.Comment
		playlists[i].SongCount = len(p.Tracks)
		playlists[i].Duration = p.Duration
		playlists[i].Owner = p.Owner
		playlists[i].Public = p.Public
	}
	response := c.NewEmpty()
	response.Playlists = &responses.Playlists{Playlist: playlists}
	c.SendResponse(response)
}

func (c *PlaylistsController) GetPlaylist() {
	id := c.RequiredParamString("id", "id parameter required")

	pinfo, err := c.pls.Get(id)
	switch {
	case err == domain.ErrNotFound:
		beego.Error(err, "Id:", id)
		c.SendError(responses.ErrorDataNotFound, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}

	response := c.NewEmpty()
	response.Playlist = c.buildPlaylist(pinfo)
	c.SendResponse(response)
}

func (c *PlaylistsController) CreatePlaylist() {
	songIds := c.RequiredParamStrings("songId", "Required parameter songId is missing")
	name := c.RequiredParamString("name", "Required parameter name is missing")
	err := c.pls.Create(name, songIds)
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}
	c.SendEmptyResponse()
}

func (c *PlaylistsController) DeletePlaylist() {
	id := c.RequiredParamString("id", "Required parameter id is missing")
	err := c.pls.Delete(id)
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}
	c.SendEmptyResponse()
}

func (c *PlaylistsController) UpdatePlaylist() {
	playlistId := c.RequiredParamString("playlistId", "Required parameter playlistId is missing")
	songsToAdd := c.ParamStrings("songIdToAdd")
	songIndexesToRemove := c.ParamInts("songIndexToRemove")

	var pname *string
	if len(c.Input()["name"]) > 0 {
		s := c.Input()["name"][0]
		pname = &s
	}

	beego.Info(fmt.Sprintf("Updating playlist with id '%s'", playlistId))
	if pname != nil {
		beego.Debug(fmt.Sprintf("-- New Name: '%s'", *pname))
	}
	beego.Debug(fmt.Sprintf("-- Adding: '%v'", songsToAdd))
	beego.Debug(fmt.Sprintf("-- Removing: '%v'", songIndexesToRemove))

	err := c.pls.Update(playlistId, pname, songsToAdd, songIndexesToRemove)
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ErrorGeneric, "Internal Error")
	}
	c.SendEmptyResponse()
}

func (c *PlaylistsController) buildPlaylist(d *engine.PlaylistInfo) *responses.PlaylistWithSongs {
	pls := &responses.PlaylistWithSongs{}
	pls.Id = d.Id
	pls.Name = d.Name
	pls.SongCount = d.SongCount
	pls.Owner = d.Owner
	pls.Duration = d.Duration
	pls.Public = d.Public

	pls.Entry = c.ToChildren(d.Entries)
	return pls
}
