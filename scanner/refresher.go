package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/core/artwork"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/navidrome/navidrome/utils/gg"
	"github.com/navidrome/navidrome/utils/slice"
	"golang.org/x/exp/maps"
)

// refresher is responsible for rolling up mediafiles attributes into albums attributes,
// and albums attributes into artists attributes. This is done by accumulating all album and artist IDs
// found during scan, and "refreshing" the albums and artists when flush is called.
//
// The actual mappings happen in MediaFiles.ToAlbum() and Albums.ToAlbumArtist()
type refresher struct {
	ds          model.DataStore
	album       map[string]struct{}
	artist      map[string]struct{}
	dirMap      dirMap
	cacheWarmer artwork.CacheWarmer
}

func newRefresher(ds model.DataStore, cw artwork.CacheWarmer, dirMap dirMap) *refresher {
	return &refresher{
		ds:          ds,
		album:       map[string]struct{}{},
		artist:      map[string]struct{}{},
		dirMap:      dirMap,
		cacheWarmer: cw,
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

func (r *refresher) flush(ctx context.Context) error {
	err := r.flushMap(ctx, r.album, "album", r.refreshAlbums)
	if err != nil {
		return err
	}
	r.album = map[string]struct{}{}
	err = r.flushMap(ctx, r.artist, "artist", r.refreshArtists)
	if err != nil {
		return err
	}
	r.artist = map[string]struct{}{}
	return nil
}

type refreshCallbackFunc = func(ctx context.Context, ids ...string) error

func (r *refresher) flushMap(ctx context.Context, m map[string]struct{}, entity string, refresh refreshCallbackFunc) error {
	if len(m) == 0 {
		return nil
	}

	ids := maps.Keys(m)
	chunks := slice.BreakUp(ids, 100)
	for _, chunk := range chunks {
		err := refresh(ctx, chunk...)
		if err != nil {
			log.Error(ctx, fmt.Sprintf("Error writing %ss to the DB", entity), err)
			return err
		}
	}
	return nil
}

func (r *refresher) refreshAlbums(ctx context.Context, ids ...string) error {
	mfs, err := r.ds.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": ids}})
	if err != nil {
		return err
	}
	if len(mfs) == 0 {
		return nil
	}

	repo := r.ds.Album(ctx)
	grouped := slice.Group(mfs, func(m model.MediaFile) string { return m.AlbumID })
	for _, group := range grouped {
		songs := model.MediaFiles(group)
		a := songs.ToAlbum()
		var updatedAt time.Time
		a.ImageFiles, updatedAt = r.getImageFiles(songs.Dirs())
		if updatedAt.After(a.UpdatedAt) {
			a.UpdatedAt = updatedAt
		}
		err := repo.Put(&a)
		if err != nil {
			return err
		}
		r.cacheWarmer.PreCache(a.CoverArtID())
	}
	return nil
}

func (r *refresher) getImageFiles(dirs []string) (string, time.Time) {
	var imageFiles []string
	var updatedAt time.Time
	for _, dir := range dirs {
		stats := r.dirMap[dir]
		for _, img := range stats.Images {
			imageFiles = append(imageFiles, filepath.Join(dir, img))
		}
		if stats.ImagesUpdatedAt.After(updatedAt) {
			updatedAt = stats.ImagesUpdatedAt
		}
	}
	return strings.Join(imageFiles, consts.Zwsp), updatedAt
}

func (r *refresher) refreshArtists(ctx context.Context, ids ...string) error {
	albums, err := r.ds.Album(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": ids}})
	if err != nil {
		return err
	}
	if len(albums) == 0 {
		return nil
	}

	repo := r.ds.Artist(ctx)
	grouped := slice.Group(albums, func(al model.Album) string { return al.AlbumArtistID })
	for _, group := range grouped {
		a := model.Albums(group).ToAlbumArtist()

		// Force a external metadata lookup on next access
		a.ExternalInfoUpdatedAt = P(time.Time{})

		// Do not remove old metadata
		err := repo.Put(&a, "album_count", "genres", "external_info_updated_at", "mbz_artist_id", "name", "order_artist_name", "size", "sort_artist_name", "song_count")
		if err != nil {
			return err
		}
		r.cacheWarmer.PreCache(a.CoverArtID())
	}
	return nil
}
