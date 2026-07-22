package persistence

import (
	"context"
	"time"

	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type itemArtworkRepository struct {
	sqlRepository
}

func NewItemArtworkRepository(ctx context.Context, db dbx.Builder) model.ItemArtworkRepository {
	r := &itemArtworkRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "item_artwork"
	return r
}

func (r *itemArtworkRepository) Get(kind, id, imageType string) (*model.ItemArtwork, error) {
	return nil, model.ErrNotFound
}

func (r *itemArtworkRepository) Put(ia *model.ItemArtwork) error {
	return nil
}

func (r *itemArtworkRepository) DeleteForItem(kind, id string) error {
	return nil
}

func (r *itemArtworkRepository) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	return nil, nil
}

func (r *itemArtworkRepository) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	return 0, nil
}

var _ model.ItemArtworkRepository = (*itemArtworkRepository)(nil)
