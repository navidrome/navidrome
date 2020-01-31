package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
)

type Browser interface {
	MediaFolders(ctx context.Context) (model.MediaFolders, error)
	Indexes(ctx context.Context, ifModifiedSince time.Time) (model.ArtistIndexes, time.Time, error)
	Directory(ctx context.Context, id string) (*DirectoryInfo, error)
	Artist(ctx context.Context, id string) (*DirectoryInfo, error)
	Album(ctx context.Context, id string) (*DirectoryInfo, error)
	GetSong(ctx context.Context, id string) (*Entry, error)
	GetGenres(ctx context.Context) (model.Genres, error)
}

func NewBrowser(ds model.DataStore) Browser {
	return &browser{ds}
}

type browser struct {
	ds model.DataStore
}

func (b *browser) MediaFolders(ctx context.Context) (model.MediaFolders, error) {
	return b.ds.MediaFolder(ctx).GetAll()
}

func (b *browser) Indexes(ctx context.Context, ifModifiedSince time.Time) (model.ArtistIndexes, time.Time, error) {
	l, err := b.ds.Property(ctx).DefaultGet(model.PropLastScan, "-1")
	ms, _ := strconv.ParseInt(l, 10, 64)
	lastModified := utils.ToTime(ms)

	if err != nil {
		return nil, time.Time{}, fmt.Errorf("error retrieving LastScan property: %v", err)
	}

	if lastModified.After(ifModifiedSince) {
		indexes, err := b.ds.Artist(ctx).GetIndex()
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
	a, albums, err := b.retrieveArtist(ctx, id)
	if err != nil {
		return nil, err
	}
	log.Debug(ctx, "Found Artist", "id", id, "name", a.Name)
	var albumIds []string
	for _, al := range albums {
		albumIds = append(albumIds, al.ID)
	}
	return b.buildArtistDir(a, albums), nil
}

func (b *browser) Album(ctx context.Context, id string) (*DirectoryInfo, error) {
	al, tracks, err := b.retrieveAlbum(ctx, id)
	if err != nil {
		return nil, err
	}
	log.Debug(ctx, "Found Album", "id", id, "name", al.Name)
	var mfIds []string
	for _, mf := range tracks {
		mfIds = append(mfIds, mf.ID)
	}

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

func (b *browser) GetSong(ctx context.Context, id string) (*Entry, error) {
	mf, err := b.ds.MediaFile(ctx).Get(id)
	if err != nil {
		return nil, err
	}

	entry := FromMediaFile(mf)
	return &entry, nil
}

func (b *browser) GetGenres(ctx context.Context) (model.Genres, error) {
	genres, err := b.ds.Genre(ctx).GetAll()
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
		Artist:     al.Artist,
		ArtistId:   al.ArtistID,
		SongCount:  al.SongCount,
		Duration:   al.Duration,
		Created:    al.CreatedAt,
		Year:       al.Year,
		Genre:      al.Genre,
		CoverArt:   al.CoverArtId,
		PlayCount:  int32(al.PlayCount),
		Starred:    al.StarredAt,
		UserRating: al.Rating,
	}

	dir.Entries = FromMediaFiles(tracks)
	return dir
}

func (b *browser) isArtist(ctx context.Context, id string) bool {
	found, err := b.ds.Artist(ctx).Exists(id)
	if err != nil {
		log.Debug(ctx, "Error searching for Artist", "id", id, err)
		return false
	}
	return found
}

func (b *browser) isAlbum(ctx context.Context, id string) bool {
	found, err := b.ds.Album(ctx).Exists(id)
	if err != nil {
		log.Debug(ctx, "Error searching for Album", "id", id, err)
		return false
	}
	return found
}

func (b *browser) retrieveArtist(ctx context.Context, id string) (a *model.Artist, as model.Albums, err error) {
	a, err = b.ds.Artist(ctx).Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Artist %s from DB: %v", id, err)
		return
	}

	if as, err = b.ds.Album(ctx).FindByArtist(id); err != nil {
		err = fmt.Errorf("Error reading %s's albums from DB: %v", a.Name, err)
	}
	return
}

func (b *browser) retrieveAlbum(ctx context.Context, id string) (al *model.Album, mfs model.MediaFiles, err error) {
	al, err = b.ds.Album(ctx).Get(id)
	if err != nil {
		err = fmt.Errorf("Error reading Album %s from DB: %v", id, err)
		return
	}

	if mfs, err = b.ds.MediaFile(ctx).FindByAlbum(id); err != nil {
		err = fmt.Errorf("Error reading %s's tracks from DB: %v", al.Name, err)
	}
	return
}
