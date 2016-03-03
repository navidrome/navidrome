package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
)

type GetMusicDirectoryController struct {
	BaseAPIController
	artistRepo domain.ArtistRepository
	albumRepo  domain.AlbumRepository
	mFileRepo  domain.MediaFileRepository
}

func (c *GetMusicDirectoryController) Prepare() {
	inject.ExtractAssignable(utils.Graph, &c.artistRepo)
	inject.ExtractAssignable(utils.Graph, &c.albumRepo)
	inject.ExtractAssignable(utils.Graph, &c.mFileRepo)
}

func (c *GetMusicDirectoryController) Get() {
	id := c.Input().Get("id")

	if id == "" {
		c.SendError(responses.ERROR_MISSING_PARAMETER, "id parameter required")
	}

	dir := &responses.Directory{}
	a, albums, found := c.retrieveArtist(id)
	if found {
		dir.Id = a.Id
		dir.Name = a.Name
		dir.Child = make([]responses.Child, len(albums))
		for i, al := range albums {
			dir.Child[i].Id = al.Id
			dir.Child[i].Title = al.Name
			dir.Child[i].IsDir = true
			dir.Child[i].Album = al.Name
			dir.Child[i].Year = al.Year
			dir.Child[i].Artist = al.Artist
			dir.Child[i].Genre = al.Genre
		}
	} else {
		al, tracks, found := c.retrieveAlbum(id)
		if found {
			dir.Id = al.Id
			dir.Name = al.Name
			dir.Child = make([]responses.Child, len(tracks))
			for i, mf := range tracks {
				dir.Child[i].Id = mf.Id
				dir.Child[i].Title = mf.Title
				dir.Child[i].IsDir = false
				dir.Child[i].Album = mf.Album
				dir.Child[i].Year = mf.Year
				dir.Child[i].Artist = mf.Artist
				dir.Child[i].Genre = mf.Genre
				dir.Child[i].Track = mf.TrackNumber
				dir.Child[i].Duration = mf.Duration
				dir.Child[i].Size = mf.Size
				dir.Child[i].Suffix = mf.Suffix
				dir.Child[i].BitRate = mf.BitRate
			}
		} else {
			beego.Info("Id", id, "not found")
			c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
		}
	}

	response := c.NewEmpty()
	response.Directory = dir
	c.SendResponse(response)
}

func (c *GetMusicDirectoryController) retrieveArtist(id string) (a *domain.Artist, as []domain.Album, found bool) {
	found, err := c.artistRepo.Exists(id)
	if err != nil {
		beego.Error("Error searching for Artist:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	if !found {
		return nil, nil, false
	}

	if a, err = c.artistRepo.Get(id); err != nil {
		beego.Error("Error reading Artist from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	if as, err = c.albumRepo.FindByArtist(id); err != nil {
		beego.Error("Error reading Album from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return
}

func (c *GetMusicDirectoryController) retrieveAlbum(id string) (al *domain.Album, mfs []domain.MediaFile, found bool) {
	found, err := c.albumRepo.Exists(id)
	if err != nil {
		beego.Error("Error searching for Album:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	if !found {
		return nil, nil, false
	}

	if al, err = c.albumRepo.Get(id); err != nil {
		beego.Error("Error reading Album from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	if mfs, err = c.mFileRepo.FindByAlbum(id); err != nil {
		beego.Error("Error reading Album from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return
}
