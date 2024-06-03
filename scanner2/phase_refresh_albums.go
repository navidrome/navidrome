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

type phaseRefreshAlbums struct {
	ds   model.DataStore
	ctx  context.Context
	libs model.Libraries
}

func createPhaseRefreshAlbums(ctx context.Context, ds model.DataStore, libs model.Libraries) *phaseRefreshAlbums {
	return &phaseRefreshAlbums{ctx: ctx, ds: ds, libs: libs}
}

func (p *phaseRefreshAlbums) producer() ppl.Producer[*model.Album] {
	return ppl.NewProducer(func(put func(album *model.Album)) error {
		for _, lib := range p.libs {
			albumIDs, err := p.ds.Album(p.ctx).GetOutdatedAlbumIDs(lib.ID)
			if err != nil {
				return fmt.Errorf("error loading outdated albums: %w", err)
			}
			if len(albumIDs) == 0 {
				continue
			}
			log.Debug(p.ctx, "Scanner: found albums needing refresh", "library_id", lib.ID, "count", len(albumIDs))
			for _, id := range albumIDs {
				put(&model.Album{ID: id})
			}
		}
		return nil
	}, ppl.Name("load albums from db"))
}

func (p *phaseRefreshAlbums) stages() []ppl.Stage[*model.Album] {
	return []ppl.Stage[*model.Album]{
		ppl.NewStage(p.refreshAlbums, ppl.Name("refresh albums")),
	}
}

func (p *phaseRefreshAlbums) refreshAlbums(album *model.Album) (*model.Album, error) {
	err := p.ds.WithTx(func(tx model.DataStore) error {
		mfs, err := tx.MediaFile(p.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": album.ID}})
		if err != nil {
			log.Error(p.ctx, "Error loading media files for album", "album_id", album.ID, err)
			return err
		}
		album = P(mfs.ToAlbum())
		log.Debug(p.ctx, "Scanner: refreshing album", "album_id", album.ID, "name", album.Name, "songCount", album.SongCount)
		return tx.Album(p.ctx).Put(album)
	})
	if err != nil {
		return nil, err
	}
	return album, nil
}

func (p *phaseRefreshAlbums) finalize() error {
	return nil
}
