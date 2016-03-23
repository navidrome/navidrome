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
	MediaFolders() (domain.MediaFolders, error)
	Indexes(ifModifiedSince time.Time) (domain.ArtistIndexes, time.Time, error)
	Directory(id string) (*DirectoryInfo, error)
}

func NewBrowser(pr PropertyRepository, fr domain.MediaFolderRepository, ir domain.ArtistIndexRepository,
	ar domain.ArtistRepository, alr domain.AlbumRepository, mr domain.MediaFileRepository) Browser {
	return &browser{pr, fr, ir, ar, alr, mr}
}

type browser struct {
	propRepo   PropertyRepository
	folderRepo domain.MediaFolderRepository
	indexRepo  domain.ArtistIndexRepository
	artistRepo domain.ArtistRepository
	albumRepo  domain.AlbumRepository
	mfileRepo  domain.MediaFileRepository
}

func (b *browser) MediaFolders() (domain.MediaFolders, error) {
	return b.folderRepo.GetAll()
}

func (b *browser) Indexes(ifModifiedSince time.Time) (domain.ArtistIndexes, time.Time, error) {
	l, err := b.propRepo.DefaultGet(consts.LastScan, "-1")
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)

	if err != nil {
		return nil, time.Time{}, errors.New(fmt.Sprintf("error retrieving LastScan property: %v", err))
	}

	if lastModified.After(ifModifiedSince) {
		indexes, err := b.indexRepo.GetAll()
		return indexes, lastModified, err
	}

	return nil, lastModified, nil
}

type DirectoryInfo struct {
	Id         string
	Name       string
	Entries    Entries
	Parent     string
	Starred    time.Time
	PlayCount  int32
	UserRating int
}

func (b *browser) Directory(id string) (*DirectoryInfo, error) {
	var dir *DirectoryInfo
	switch {
	case b.isArtist(id):
		beego.Debug("Found Artist with id", id)
		a, albums, err := b.retrieveArtist(id)
		if err != nil {
			return nil, err
		}
		dir = b.buildArtistDir(a, albums)
	case b.isAlbum(id):
		beego.Debug("Found Album with id", id)
		al, tracks, err := b.retrieveAlbum(id)
		if err != nil {
			return nil, err
		}
		dir = b.buildAlbumDir(al, tracks)
	default:
		beego.Debug("Id", id, "not found")
		return nil, ErrDataNotFound
	}

	return dir, nil
}

func (b *browser) buildArtistDir(a *domain.Artist, albums domain.Albums) *DirectoryInfo {
	dir := &DirectoryInfo{Id: a.Id, Name: a.Name}

	dir.Entries = make(Entries, len(albums))
	for i, al := range albums {
		dir.Entries[i] = FromAlbum(&al)
		dir.PlayCount += int32(al.PlayCount)
	}
	return dir
}

func (b *browser) buildAlbumDir(al *domain.Album, tracks domain.MediaFiles) *DirectoryInfo {
	dir := &DirectoryInfo{
		Id:         al.Id,
		Name:       al.Name,
		Parent:     al.ArtistId,
		PlayCount:  int32(al.PlayCount),
		UserRating: al.Rating,
		Starred:    al.StarredAt,
	}

	dir.Entries = make(Entries, len(tracks))
	for i, mf := range tracks {
		dir.Entries[i] = FromMediaFile(&mf)
	}
	return dir
}

func (b *browser) isArtist(id string) bool {
	found, err := b.artistRepo.Exists(id)
	if err != nil {
		beego.Debug(fmt.Errorf("Error searching for Artist %s: %v", id, err))
		return false
	}
	return found
}

func (b *browser) isAlbum(id string) bool {
	found, err := b.albumRepo.Exists(id)
	if err != nil {
		beego.Debug(fmt.Errorf("Error searching for Album %s: %v", id, err))
		return false
	}
	return found
}

func (b *browser) retrieveArtist(id string) (a *domain.Artist, as domain.Albums, err error) {
	a, err = b.artistRepo.Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Artist %s from DB: %v", id, err)
		return
	}

	if as, err = b.albumRepo.FindByArtist(id); err != nil {
		err = fmt.Errorf("Error reading %s's albums from DB: %v", a.Name, err)
	}
	return
}

func (b *browser) retrieveAlbum(id string) (al *domain.Album, mfs domain.MediaFiles, err error) {
	al, err = b.albumRepo.Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Album %s from DB: %v", id, err)
		return
	}

	if mfs, err = b.mfileRepo.FindByAlbum(id); err != nil {
		err = fmt.Errorf("Error reading %s's tracks from DB: %v", al.Name, err)
	}
	return
}
