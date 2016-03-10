package engine

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/astaxie/beego"
	"github.com/deluan/gosonic/consts"
	"github.com/deluan/gosonic/domain"
	"github.com/deluan/gosonic/utils"
)

type Browser interface {
	MediaFolders() (*domain.MediaFolders, error)
	Indexes(ifModifiedSince time.Time) (*domain.ArtistIndexes, time.Time, error)
	Directory(id string) (*DirectoryInfo, error)
}

func NewBrowser(pr PropertyRepository, fr domain.MediaFolderRepository, ir domain.ArtistIndexRepository,
	ar domain.ArtistRepository, alr domain.AlbumRepository, mr domain.MediaFileRepository) Browser {
	return browser{pr, fr, ir, ar, alr, mr}
}

type browser struct {
	propRepo   PropertyRepository
	folderRepo domain.MediaFolderRepository
	indexRepo  domain.ArtistIndexRepository
	artistRepo domain.ArtistRepository
	albumRepo  domain.AlbumRepository
	mfileRepo  domain.MediaFileRepository
}

func (b browser) MediaFolders() (*domain.MediaFolders, error) {
	return b.folderRepo.GetAll()
}

func (b browser) Indexes(ifModifiedSince time.Time) (*domain.ArtistIndexes, time.Time, error) {
	l, err := b.propRepo.DefaultGet(consts.LastScan, "-1")
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)

	if err != nil {
		return &domain.ArtistIndexes{}, time.Time{}, errors.New(fmt.Sprintf("Error retrieving LastScan property: %v", err))
	}

	if lastModified.After(ifModifiedSince) {
		indexes, err := b.indexRepo.GetAll()
		return indexes, lastModified, err
	}

	return &domain.ArtistIndexes{}, lastModified, nil
}

type DirectoryInfo struct {
	Id       string
	Name     string
	Children []Child
}

func (c browser) Directory(id string) (*DirectoryInfo, error) {
	var dir *DirectoryInfo
	switch {
	case c.isArtist(id):
		beego.Debug("Found Artist with id", id)
		a, albums, err := c.retrieveArtist(id)
		if err != nil {
			return nil, err
		}
		dir = c.buildArtistDir(a, albums)
	case c.isAlbum(id):
		beego.Debug("Found Album with id", id)
		al, tracks, err := c.retrieveAlbum(id)
		if err != nil {
			return nil, err
		}
		dir = c.buildAlbumDir(al, tracks)
	default:
		beego.Debug("Id", id, "not found")
		return nil, ErrDataNotFound
	}

	return dir, nil
}

func (c browser) buildArtistDir(a *domain.Artist, albums *domain.Albums) *DirectoryInfo {
	dir := &DirectoryInfo{Id: a.Id, Name: a.Name}

	dir.Children = make([]Child, len(*albums))
	for i, al := range *albums {
		dir.Children[i].Id = al.Id
		dir.Children[i].Title = al.Name
		dir.Children[i].IsDir = true
		dir.Children[i].Parent = al.ArtistId
		dir.Children[i].Album = al.Name
		dir.Children[i].Year = al.Year
		dir.Children[i].Artist = al.AlbumArtist
		dir.Children[i].Genre = al.Genre
		dir.Children[i].CoverArt = al.CoverArtId
		if al.Starred {
			dir.Children[i].Starred = al.UpdatedAt
		}

	}
	return dir
}

func (c browser) buildAlbumDir(al *domain.Album, tracks *domain.MediaFiles) *DirectoryInfo {
	dir := &DirectoryInfo{Id: al.Id, Name: al.Name}

	dir.Children = make([]Child, len(*tracks))
	for i, mf := range *tracks {
		dir.Children[i].Id = mf.Id
		dir.Children[i].Title = mf.Title
		dir.Children[i].IsDir = false
		dir.Children[i].Parent = mf.AlbumId
		dir.Children[i].Album = mf.Album
		dir.Children[i].Year = mf.Year
		dir.Children[i].Artist = mf.Artist
		dir.Children[i].Genre = mf.Genre
		dir.Children[i].Track = mf.TrackNumber
		dir.Children[i].Duration = mf.Duration
		dir.Children[i].Size = mf.Size
		dir.Children[i].Suffix = mf.Suffix
		dir.Children[i].BitRate = mf.BitRate
		if mf.Starred {
			dir.Children[i].Starred = mf.UpdatedAt
		}
		if mf.HasCoverArt {
			dir.Children[i].CoverArt = mf.Id
		}
		dir.Children[i].ContentType = mf.ContentType()
	}
	return dir
}

func (c browser) isArtist(id string) bool {
	found, err := c.artistRepo.Exists(id)
	if err != nil {
		beego.Debug(fmt.Errorf("Error searching for Artist %s: %v", id, err))
		return false
	}
	return found
}

func (c browser) isAlbum(id string) bool {
	found, err := c.albumRepo.Exists(id)
	if err != nil {
		beego.Debug(fmt.Errorf("Error searching for Album %s: %v", id, err))
		return false
	}
	return found
}

func (c browser) retrieveArtist(id string) (a *domain.Artist, as *domain.Albums, err error) {
	a, err = c.artistRepo.Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Artist %s from DB: %v", id, err)
		return
	}

	if as, err = c.albumRepo.FindByArtist(id); err != nil {
		err = fmt.Errorf("Error reading %s's albums from DB: %v", a.Name, err)
	}
	return
}

func (c browser) retrieveAlbum(id string) (al *domain.Album, mfs *domain.MediaFiles, err error) {
	al, err = c.albumRepo.Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Album %s from DB: %v", id, err)
		return
	}

	if mfs, err = c.mfileRepo.FindByAlbum(id); err != nil {
		err = fmt.Errorf("Error reading %s's tracks from DB: %v", al.Name, err)
	}
	return
}
