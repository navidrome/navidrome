package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type artworkQueueRepository struct {
	sqlRepository
}

func NewArtworkQueueRepository(ctx context.Context, db dbx.Builder) model.ArtworkQueueRepository {
	r := &artworkQueueRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork_queue"
	return r
}

func (r *artworkQueueRepository) Enqueue(items ...model.ArtworkQueueItem) error {
	now := time.Now()
	for _, it := range items {
		if it.ImageType == "" {
			it.ImageType = model.ImageTypePrimary
		}
		ins := Insert(r.tableName).SetMap(map[string]any{
			"item_kind": it.ItemKind, "item_id": it.ItemID, "image_type": it.ImageType,
			"priority": it.Priority, "attempts": 0, "retry_at": now, "enqueued_at": now,
		}).Suffix(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
			priority = MAX(priority, excluded.priority), retry_at = excluded.retry_at`)
		if _, err := r.executeSQL(ins); err != nil {
			return err
		}
	}
	return nil
}

func (r *artworkQueueRepository) DequeueBatch(n int) ([]model.ArtworkQueueItem, error) {
	sel := Select("*").From(r.tableName).
		Where(LtOrEq{"retry_at": time.Now()}).
		OrderBy("priority DESC", "enqueued_at ASC").
		Limit(uint64(n))
	var res []model.ArtworkQueueItem
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *artworkQueueRepository) MarkFailed(kind, id, imageType string, retryAt time.Time) error {
	upd := Update(r.tableName).
		Set("attempts", Expr("attempts + 1")).
		Set("retry_at", retryAt).
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
	c, err := r.executeSQL(upd)
	if err == nil && c == 0 {
		return model.ErrNotFound
	}
	return err
}

func (r *artworkQueueRepository) Delete(kind, id, imageType string) error {
	return r.delete(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
}

func (r *artworkQueueRepository) Count() (int64, error) {
	sel := Select("count(*)").From(r.tableName)
	var counts []int64
	if err := r.queryAllSlice(sel, &counts); err != nil {
		return 0, err
	}
	if len(counts) == 0 {
		return 0, nil
	}
	return counts[0], nil
}

var _ model.ArtworkQueueRepository = (*artworkQueueRepository)(nil)
