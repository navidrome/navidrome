package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/navidrome/navidrome/utils/slice"
)

type refresher struct {
	ctx    context.Context
	ds     model.DataStore
	album  map[string]struct{}
	artist map[string]struct{}
	dirMap dirMap
}

func newRefresher(ctx context.Context, ds model.DataStore, dirMap dirMap) *refresher {
	return &refresher{
		ctx:    ctx,
		ds:     ds,
		album:  map[string]struct{}{},
		artist: map[string]struct{}{},
		dirMap: dirMap,
	}
}

func (f *refresher) accumulate(mf model.MediaFile) {
	if mf.AlbumID != "" {
		f.album[mf.AlbumID] = struct{}{}
	}
	if mf.AlbumArtistID != "" {
		f.artist[mf.AlbumArtistID] = struct{}{}
	}
}

type refreshCallbackFunc = func(ids ...string) error

func (f *refresher) flushMap(m map[string]struct{}, entity string, refresh refreshCallbackFunc) error {
	if len(m) == 0 {
		return nil
	}
	var ids []string
	for id := range m {
		ids = append(ids, id)
		delete(m, id)
	}
	if err := refresh(ids...); err != nil {
		log.Error(f.ctx, fmt.Sprintf("Error writing %ss to the DB", entity), err)
		return err
	}
	return nil
}

func (f *refresher) refreshAlbumsChunked(ids ...string) error {
	chunks := utils.BreakUpStringSlice(ids, 100)
	for _, chunk := range chunks {
		err := f.refreshAlbums(chunk...)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *refresher) refreshAlbums(ids ...string) error {
	mfs, err := f.ds.MediaFile(f.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": ids}})
	if err != nil {
		return err
	}
	if len(mfs) == 0 {
		return nil
	}

	repo := f.ds.Album(f.ctx)
	grouped := slice.Group(mfs, func(m model.MediaFile) string { return m.AlbumID })
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		a := songs.ToAlbum()
		a.ImageFiles = f.getImageFiles(songs.Dirs())
		err := repo.Put(&a)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *refresher) getImageFiles(dirs []string) string {
	var imageFiles []string
	for _, dir := range dirs {
		for _, img := range f.dirMap[dir].Images {
			imageFiles = append(imageFiles, filepath.Join(dir, img))
		}
	}
	return strings.Join(imageFiles, string(filepath.ListSeparator))
}

func (f *refresher) flush() error {
	err := f.flushMap(f.album, "album", f.refreshAlbumsChunked)
	if err != nil {
		return err
	}
	err = f.flushMap(f.artist, "artist", f.ds.Artist(f.ctx).Refresh) // TODO Move Artist Refresh out of persistence
	if err != nil {
		return err
	}
	return nil
}
