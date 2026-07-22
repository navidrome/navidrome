package persistence

import (
	"context"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

// enqueueChunkSize keeps each multi-row insert under SQLite's bind-variable limit (7 cols -> 700 vars).
const enqueueChunkSize = 100

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
	for chunk := range slices.Chunk(items, enqueueChunkSize) {
		ins := Insert(r.tableName).Columns("item_kind", "item_id", "image_type", "priority", "attempts", "retry_at", "enqueued_at")
		for _, it := range chunk {
			if it.ImageType == "" {
				it.ImageType = model.ImageTypePrimary
			}
			ins = ins.Values(it.ItemKind, it.ItemID, it.ImageType, it.Priority, 0, now, now)
		}
		ins = ins.Suffix(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
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

// MarkFailedIfUnchanged applies the backoff only while retry_at still equals seenRetryAt;
// a concurrent Enqueue resets retry_at, so its fresh eligibility survives untouched.
func (r *artworkQueueRepository) MarkFailedIfUnchanged(kind, id, imageType string, seenRetryAt, retryAt time.Time) error {
	upd := Update(r.tableName).
		Set("attempts", Expr("attempts + 1")).
		Set("retry_at", retryAt).
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType, "retry_at": seenRetryAt})
	_, err := r.executeSQL(upd)
	return err
}

func (r *artworkQueueRepository) Delete(kind, id, imageType string) error {
	return r.delete(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
}

// DeleteIfUnchanged deletes the row only while its retry_at still equals the dequeued
// value; a concurrent Enqueue resets retry_at, so the row survives to be re-resolved.
func (r *artworkQueueRepository) DeleteIfUnchanged(kind, id, imageType string, retryAt time.Time) error {
	return r.delete(Eq{"item_kind": kind, "item_id": id, "image_type": imageType, "retry_at": retryAt})
}

// PurgeDangling removes queue rows whose entity no longer exists, per kind.
func (r *artworkQueueRepository) PurgeDangling() (int64, error) {
	return purgeDangling(r.executeSQL, r.tableName)
}

func (r *artworkQueueRepository) Count() (int64, error) {
	var res struct{ Count int64 }
	err := r.queryOne(Select("count(*) as count").From(r.tableName), &res)
	return res.Count, err
}

func (r *artworkQueueRepository) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	now := time.Now()
	// DO NOTHING is deliberate: rechecks must not bump priority/retry_at of already-queued items.
	ins := Expr(`INSERT INTO `+r.tableName+` (item_kind, item_id, image_type, priority, attempts, retry_at, enqueued_at)
		SELECT item_kind, item_id, image_type, ?, 0, ?, ?
		FROM `+itemArtworkTable+` WHERE item_kind = ? AND hash = '' AND attempted_at < ?
		ON CONFLICT (item_kind, item_id, image_type) DO NOTHING`,
		model.ArtworkPriorityRecheck, now, now, kind, attemptedBefore)
	return r.executeSQL(ins)
}

var _ model.ArtworkQueueRepository = (*artworkQueueRepository)(nil)
