package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/utils"
)

type Browser interface {
	MediaFolders() (model.MediaFolders, error)
	Indexes(ifModifiedSince time.Time) (model.ArtistIndexes, time.Time, error)
	Directory(ctx context.Context, id string) (*DirectoryInfo, error)
	Artist(ctx context.Context, id string) (*DirectoryInfo, error)
	Album(ctx context.Context, id string) (*DirectoryInfo, error)
	GetSong(id string) (*Entry, error)
	GetGenres() (model.Genres, error)
}

func NewBrowser(pr model.PropertyRepository, fr model.MediaFolderRepository,
	ar model.ArtistRepository, alr model.AlbumRepository, mr model.MediaFileRepository, gr model.GenreRepository) Browser {
	return &browser{pr, fr, ar, alr, mr, gr}
}

type browser struct {
	propRepo   model.PropertyRepository
	folderRepo model.MediaFolderRepository
	artistRepo model.ArtistRepository
	albumRepo  model.AlbumRepository
	mfileRepo  model.MediaFileRepository
	genreRepo  model.GenreRepository
}

func (b *browser) MediaFolders() (model.MediaFolders, error) {
	return b.folderRepo.GetAll()
}

func (b *browser) Indexes(ifModifiedSince time.Time) (model.ArtistIndexes, time.Time, error) {
	l, err := b.propRepo.DefaultGet(model.PropLastScan, "-1")
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)

	if err != nil {
		return nil, time.Time{}, fmt.Errorf("error retrieving LastScan property: %v", err)
	}

	if lastModified.After(ifModifiedSince) {
		indexes, err := b.artistRepo.GetIndex()
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
	AlbumCount int
	CoverArt   string
	Artist     string
	ArtistId   string
	SongCount  int
	Duration   int
	Created    time.Time
	Year       int
	Genre      string
}

func (b *browser) Artist(ctx context.Context, id string) (*DirectoryInfo, error) {
	a, albums, err := b.retrieveArtist(id)
	if err != nil {
		return nil, err
	}
	log.Debug(ctx, "Found Artist", "id", id, "name", a.Name)
	return b.buildArtistDir(a, albums), nil
}

func (b *browser) Album(ctx context.Context, id string) (*DirectoryInfo, error) {
	al, tracks, err := b.retrieveAlbum(id)
	if err != nil {
		return nil, err
	}
	log.Debug(ctx, "Found Album", "id", id, "name", al.Name)
	return b.buildAlbumDir(al, tracks), nil
}

func (b *browser) Directory(ctx context.Context, id string) (*DirectoryInfo, error) {
	switch {
	case b.isArtist(ctx, id):
		return b.Artist(ctx, id)
	case b.isAlbum(ctx, id):
		return b.Album(ctx, id)
	default:
		log.Debug(ctx, "Directory not found", "id", id)
		return nil, model.ErrNotFound
	}
}

func (b *browser) GetSong(id string) (*Entry, error) {
	mf, err := b.mfileRepo.Get(id)
	if err != nil {
		return nil, err
	}

	entry := FromMediaFile(mf)
	return &entry, nil
}

func (b *browser) GetGenres() (model.Genres, error) {
	genres, err := b.genreRepo.GetAll()
	for i, g := range genres {
		if strings.TrimSpace(g.Name) == "" {
			genres[i].Name = "<Empty>"
		}
	}
	sort.Slice(genres, func(i, j int) bool {
		return genres[i].Name < genres[j].Name
	})
	return genres, err
}

func (b *browser) buildArtistDir(a *model.Artist, albums model.Albums) *DirectoryInfo {
	dir := &DirectoryInfo{
		Id:         a.ID,
		Name:       a.Name,
		AlbumCount: a.AlbumCount,
	}

	dir.Entries = make(Entries, len(albums))
	for i, al := range albums {
		dir.Entries[i] = FromAlbum(&al)
		dir.PlayCount += int32(al.PlayCount)
	}
	return dir
}

func (b *browser) buildAlbumDir(al *model.Album, tracks model.MediaFiles) *DirectoryInfo {
	dir := &DirectoryInfo{
		Id:         al.ID,
		Name:       al.Name,
		Parent:     al.ArtistID,
		PlayCount:  int32(al.PlayCount),
		UserRating: al.Rating,
		Starred:    al.StarredAt,
		Artist:     al.Artist,
		ArtistId:   al.ArtistID,
		SongCount:  al.SongCount,
		Duration:   al.Duration,
		Created:    al.CreatedAt,
		Year:       al.Year,
		Genre:      al.Genre,
		CoverArt:   al.CoverArtId,
	}

	dir.Entries = make(Entries, len(tracks))
	for i, mf := range tracks {
		dir.Entries[i] = FromMediaFile(&mf)
	}
	return dir
}

func (b *browser) isArtist(ctx context.Context, id string) bool {
	found, err := b.artistRepo.Exists(id)
	if err != nil {
		log.Debug(ctx, "Error searching for Artist", "id", id, err)
		return false
	}
	return found
}

func (b *browser) isAlbum(ctx context.Context, id string) bool {
	found, err := b.albumRepo.Exists(id)
	if err != nil {
		log.Debug(ctx, "Error searching for Album", "id", id, err)
		return false
	}
	return found
}

func (b *browser) retrieveArtist(id string) (a *model.Artist, as model.Albums, err error) {
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

func (b *browser) retrieveAlbum(id string) (al *model.Album, mfs model.MediaFiles, err error) {
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
