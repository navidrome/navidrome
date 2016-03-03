package api

import (
	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/api/responses"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
	"github.com/karlkfi/inject"
	"time"
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
	id := c.GetParameter("id", "id parameter required")

	response := c.NewEmpty()

	switch {
	case c.isArtist(id):
		a, albums := c.retrieveArtist(id)
		response.Directory = c.buildArtistDir(a, albums)
	case c.isAlbum(id):
		al, tracks := c.retrieveAlbum(id)
		response.Directory = c.buildAlbumDir(al, tracks)
	default:
		beego.Info("Id", id, "not found")
		c.SendError(responses.ERROR_DATA_NOT_FOUND, "Directory not found")
	}

	c.SendResponse(response)
}

func (c *GetMusicDirectoryController) buildArtistDir(a *domain.Artist, albums []domain.Album) *responses.Directory {
	dir := &responses.Directory{Id: a.Id, Name: a.Name}

	dir.Child = make([]responses.Child, len(albums))
	for i, al := range albums {
		dir.Child[i].Id = al.Id
		dir.Child[i].Title = al.Name
		dir.Child[i].IsDir = true
		dir.Child[i].Album = al.Name
		dir.Child[i].Year = al.Year
		dir.Child[i].Artist = al.Artist
		dir.Child[i].Genre = al.Genre
		dir.Child[i].CoverArt = al.CoverArtId
		if al.Starred {
			t := time.Now()
			dir.Child[i].Starred = &t
		}

	}
	return dir
}

func (c *GetMusicDirectoryController) buildAlbumDir(al *domain.Album, tracks []domain.MediaFile) *responses.Directory {
	dir := &responses.Directory{Id: al.Id, Name: al.Name}

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
		if mf.Starred {
			dir.Child[i].Starred = &mf.UpdatedAt
		}
		if mf.HasCoverArt {
			dir.Child[i].CoverArt = mf.Id
		}
		dir.Child[i].ContentType = mf.ContentType()
	}
	return dir
}

func (c *GetMusicDirectoryController) isArtist(id string) bool {
	found, err := c.artistRepo.Exists(id)
	if err != nil {
		beego.Error("Error searching for Artist:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return found
}

func (c *GetMusicDirectoryController) isAlbum(id string) bool {
	found, err := c.albumRepo.Exists(id)
	if err != nil {
		beego.Error("Error searching for Album:", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return found
}

func (c *GetMusicDirectoryController) retrieveArtist(id string) (a *domain.Artist, as []domain.Album) {
	a, err := c.artistRepo.Get(id)
	if err != nil {
		beego.Error("Error reading Artist from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	if as, err = c.albumRepo.FindByArtist(id); err != nil {
		beego.Error("Error reading Artist's Albums from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return
}

func (c *GetMusicDirectoryController) retrieveAlbum(id string) (al *domain.Album, mfs []domain.MediaFile) {
	al, err := c.albumRepo.Get(id)
	if err != nil {
		beego.Error("Error reading Album from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}

	if mfs, err = c.mFileRepo.FindByAlbum(id); err != nil {
		beego.Error("Error reading Album's Tracks from DB", err)
		c.SendError(responses.ERROR_GENERIC, "Internal Error")
	}
	return
}
