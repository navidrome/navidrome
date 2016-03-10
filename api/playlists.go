package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/engine"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type PlaylistsController struct {
	BaseAPIController
	pls engine.Playlists
}

func (c *PlaylistsController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.pls)
}

func (c *PlaylistsController) GetAll() {
	allPls, err := c.pls.GetAll()
	if err != nil {
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal error")
	}
	playlists := make([]responses.Playlist, len(*allPls))
	for i, f := range *allPls {
		playlists[i].Id = f.Id
		playlists[i].Name = f.Name
		playlists[i].Comment = "Original: " + f.FullPath
		playlists[i].SongCount = len(f.Tracks)
	}
	response := c.NewEmpty()
	response.Playlists = &responses.Playlists{Playlist: playlists}
	c.SendResponse(response)
}

func (c *PlaylistsController) Get() {
	id := c.RequiredParamString("id", "id parameter required")

	pinfo, err := c.pls.Get(id)
	switch {
	case err == engine.ErrDataNotFound:
		beego.Error(err, "Id:", id)
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	case err != nil:
		beego.Error(err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	response := c.NewEmpty()
	response.Playlist = c.buildPlaylist(pinfo)
	c.SendResponse(response)
}

func (c *PlaylistsController) buildPlaylist(d *engine.PlaylistInfo) *responses.PlaylistWithSongs {
	pls := &responses.PlaylistWithSongs{}
	pls.Id = d.Id
	pls.Name = d.Name

	pls.Entry = make([]responses.Child, len(d.Entries))
	for i, child := range d.Entries {
		pls.Entry[i].Id = child.Id
		pls.Entry[i].Title = child.Title
		pls.Entry[i].IsDir = child.IsDir
		pls.Entry[i].Parent = child.Parent
		pls.Entry[i].Album = child.Album
		pls.Entry[i].Year = child.Year
		pls.Entry[i].Artist = child.Artist
		pls.Entry[i].Genre = child.Genre
		pls.Entry[i].CoverArt = child.CoverArt
		pls.Entry[i].Track = child.Track
		pls.Entry[i].Duration = child.Duration
		pls.Entry[i].Size = child.Size
		pls.Entry[i].Suffix = child.Suffix
		pls.Entry[i].BitRate = child.BitRate
		pls.Entry[i].ContentType = child.ContentType
		if !child.Starred.IsZero() {
			pls.Entry[i].Starred = &child.Starred
		}
	}
	return pls
}
