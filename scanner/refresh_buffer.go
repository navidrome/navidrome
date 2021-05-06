package scanner

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type refreshBuffer struct {
	ctx       context.Context
	ds        model.DataStore
	album     map[string]struct{}
	artist    map[string]struct{}
	genre     map[string]struct{}
	genretype map[string]struct{}
}

func newRefreshBuffer(ctx context.Context, ds model.DataStore) *refreshBuffer {
	return &refreshBuffer{
		ctx:    ctx,
		ds:     ds,
		album:  map[string]struct{}{},
		artist: map[string]struct{}{},
	}
}

func (f *refreshBuffer) accumulate(mf model.MediaFile) {
	if mf.AlbumID != "" {
		f.album[mf.AlbumID] = struct{}{}
	}
	if mf.AlbumArtistID != "" {
		f.artist[mf.AlbumArtistID] = struct{}{}
	}
	if mf.Genre != "" {
		f.genre[mf.Genre] = struct{}{}
		f.genretype[mf.Genre] = struct{}{}
	}
}

type refreshCallbackFunc = func(ids ...string) error

func (f *refreshBuffer) flushMap(m map[string]struct{}, entity string, refresh refreshCallbackFunc) error {
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

func (f *refreshBuffer) flush() error {
	err := f.flushMap(f.album, "album", f.ds.Album(f.ctx).Refresh)
	if err != nil {
		return err
	}
	err = f.flushMap(f.artist, "artist", f.ds.Artist(f.ctx).Refresh)
	if err != nil {
		return err
	}
	err = f.flushMap(f.genre, "genre", f.ds.Genre(f.ctx).Refresh)
	if err != nil {
		return err
	}
	err = f.flushMap(f.genretype, "genre_type", f.ds.GenreType(f.ctx).Refresh)
	if err != nil {
		return err
	}
	return nil
}
