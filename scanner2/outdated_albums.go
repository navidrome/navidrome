package scanner2

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	ppl "github.com/google/go-pipeline/pkg/pipeline"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	. "github.com/navidrome/navidrome/utils/gg"
)

func produceOutdatedAlbums(ctx context.Context, ds model.DataStore, libs model.Libraries) ppl.ProducerFn[*model.Album] {
	return func(put func(album *model.Album)) error {
		for _, lib := range libs {
			albumIDs, err := ds.Album(ctx).GetOutdatedAlbumIDs(lib.ID)
			if err != nil {
				return fmt.Errorf("error loading outdated albums: %w", err)
			}
			if len(albumIDs) == 0 {
				continue
			}
			log.Debug(ctx, "Scanner: found albums needing refresh", "library_id", lib.ID, "count", len(albumIDs))
			for _, id := range albumIDs {
				put(&model.Album{ID: id})
			}
		}
		return nil
	}
}

func refreshAlbums(ctx context.Context, ds model.DataStore) ppl.StageFn[*model.Album] {
	return func(album *model.Album) (*model.Album, error) {
		err := ds.WithTx(func(tx model.DataStore) error {
			mfs, err := tx.MediaFile(ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": album.ID}})
			if err != nil {
				log.Error(ctx, "Error loading media files for album", "album_id", album.ID, err)
				return err
			}
			album = P(mfs.ToAlbum())
			log.Debug(ctx, "Scanner: refreshing album", "album_id", album.ID, "name", album.Name, "songCount", album.SongCount)
			return tx.Album(ctx).Put(album)
		})
		if err != nil {
			return nil, err
		}
		return album, nil
	}
}
