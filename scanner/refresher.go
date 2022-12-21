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

// refresher is responsible for rolling up mediafiles attributes into albums attributes,
// and albums attributes into artists attributes. This is done by accumulating all album and artist IDs
// found during scan, and "refreshing" the albums and artists when flush is called.
//
// The actual mappings happen in MediaFiles.ToAlbum() and Albums.ToAlbumArtist()
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

func (r *refresher) accumulate(mf model.MediaFile) {
	if mf.AlbumID != "" {
		r.album[mf.AlbumID] = struct{}{}
	}
	if mf.AlbumArtistID != "" {
		r.artist[mf.AlbumArtistID] = struct{}{}
	}
}

func (r *refresher) flush() error {
	err := r.flushMap(r.album, "album", r.refreshAlbums)
	if err != nil {
		return err
	}
	err = r.flushMap(r.artist, "artist", r.refreshArtists)
	if err != nil {
		return err
	}
	return nil
}

type refreshCallbackFunc = func(ids ...string) error

func (r *refresher) flushMap(m map[string]struct{}, entity string, refresh refreshCallbackFunc) error {
	if len(m) == 0 {
		return nil
	}
	var ids []string
	for id := range m {
		ids = append(ids, id)
		delete(m, id)
	}
	chunks := utils.BreakUpStringSlice(ids, 100)
	for _, chunk := range chunks {
		err := refresh(chunk...)
		if err != nil {
			log.Error(r.ctx, fmt.Sprintf("Error writing %ss to the DB", entity), err)
			return err
		}
	}
	return nil
}

func (r *refresher) refreshAlbums(ids ...string) error {
	mfs, err := r.ds.MediaFile(r.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": ids}})
	if err != nil {
		return err
	}
	if len(mfs) == 0 {
		return nil
	}

	repo := r.ds.Album(r.ctx)
	grouped := slice.Group(mfs, func(m model.MediaFile) string { return m.AlbumID })
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		a := songs.ToAlbum()
		a.ImageFiles = r.getImageFiles(songs.Dirs())
		err := repo.Put(&a)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *refresher) getImageFiles(dirs []string) string {
	var imageFiles []string
	for _, dir := range dirs {
		for _, img := range r.dirMap[dir].Images {
			imageFiles = append(imageFiles, filepath.Join(dir, img))
		}
	}
	return strings.Join(imageFiles, string(filepath.ListSeparator))
}

func (r *refresher) refreshArtists(ids ...string) error {
	albums, err := r.ds.Album(r.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": ids}})
	if err != nil {
		return err
	}
	if len(albums) == 0 {
		return nil
	}

	repo := r.ds.Artist(r.ctx)
	grouped := slice.Group(albums, func(al model.Album) string { return al.AlbumArtistID })
	for _, group := range grouped {
		a := model.Albums(group).ToAlbumArtist()
		err := repo.Put(&a)
		if err != nil {
			return err
		}
	}
	return nil
}
