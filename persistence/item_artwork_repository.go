package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
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
	sel := Select("*").From(r.tableName).
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
	var res model.ItemArtwork
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *itemArtworkRepository) Put(ia *model.ItemArtwork) error {
	if ia.ImageType == "" {
		ia.ImageType = model.ImageTypePrimary
	}
	ia.UpdatedAt = time.Now()
	ins := Insert(r.tableName).SetMap(map[string]any{
		"item_kind": ia.ItemKind, "item_id": ia.ItemID, "image_type": ia.ImageType,
		"hash": ia.Hash, "source": ia.Source,
		"attempted_at": ia.AttemptedAt, "updated_at": ia.UpdatedAt,
	}).Suffix(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
		hash=excluded.hash, source=excluded.source,
		attempted_at=excluded.attempted_at, updated_at=excluded.updated_at`)
	_, err := r.executeSQL(ins)
	return err
}

func (r *itemArtworkRepository) DeleteForItem(kind, id string) error {
	return r.delete(Eq{"item_kind": kind, "item_id": id})
}

func (r *itemArtworkRepository) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	res := map[string]model.ItemArtworkInfo{}
	if len(ids) == 0 {
		return res, nil
	}
	sel := Select("ia.item_id", "ia.hash", "COALESCE(a.blur_hash, '') as blur_hash").
		From(r.tableName + " ia").
		LeftJoin("artwork a ON a.hash = ia.hash").
		Where(And{
			Eq{"ia.item_kind": kind},
			Eq{"ia.image_type": model.ImageTypePrimary},
			Eq{"ia.item_id": ids},
		})
	var rows []struct {
		ItemID   string
		Hash     string
		BlurHash string
	}
	if err := r.queryAll(sel, &rows); err != nil {
		return nil, err
	}
	for _, row := range rows {
		res[row.ItemID] = model.ItemArtworkInfo{
			ItemID: row.ItemID, Hash: row.Hash, BlurHash: row.BlurHash, Absent: row.Hash == "",
		}
	}
	return res, nil
}

func (r *itemArtworkRepository) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	now := time.Now()
	ins := Expr(`INSERT INTO artwork_queue (item_kind, item_id, image_type, priority, attempts, retry_at, enqueued_at)
		SELECT item_kind, item_id, image_type, ?, 0, ?, ?
		FROM item_artwork WHERE item_kind = ? AND hash = '' AND attempted_at < ?
		ON CONFLICT (item_kind, item_id, image_type) DO NOTHING`,
		model.ArtworkPriorityRecheck, now, now, kind, attemptedBefore)
	return r.executeSQL(ins)
}

var _ model.ItemArtworkRepository = (*itemArtworkRepository)(nil)
