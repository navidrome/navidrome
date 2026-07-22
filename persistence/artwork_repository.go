package persistence

import (
	"context"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type artworkRepository struct {
	sqlRepository
}

func NewArtworkRepository(ctx context.Context, db dbx.Builder) model.ArtworkRepository {
	r := &artworkRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork"
	return r
}

func (r *artworkRepository) GetImage(hash string) (*model.Artwork, error) {
	sel := Select("*").From(r.tableName).Where(Eq{"hash": hash})
	var res model.Artwork
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *artworkRepository) PutImage(a *model.Artwork) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now()
	}
	ins := Insert(r.tableName).SetMap(map[string]any{
		"hash": a.Hash, "mime": a.Mime, "width": a.Width, "height": a.Height,
		"size_bytes": a.SizeBytes, "blur_hash": a.BlurHash,
		"source_path": a.SourcePath, "ref_mtime": a.RefMtime, "created_at": a.CreatedAt,
	}).Suffix(`ON CONFLICT (hash) DO UPDATE SET mime=excluded.mime, width=excluded.width,
		height=excluded.height, size_bytes=excluded.size_bytes, blur_hash=excluded.blur_hash,
		source_path=excluded.source_path, ref_mtime=excluded.ref_mtime`)
	_, err := r.executeSQL(ins)
	return err
}

func (r *artworkRepository) GetImages(hashes []string) (map[string]model.Artwork, error) {
	res := map[string]model.Artwork{}
	for chunk := range slices.Chunk(hashes, 200) {
		sel := Select("*").From(r.tableName).Where(Eq{"hash": chunk})
		var all []model.Artwork
		if err := r.queryAll(sel, &all); err != nil {
			return nil, err
		}
		for _, a := range all {
			res[a.Hash] = a
		}
	}
	return res, nil
}

func (r *artworkRepository) GetOrphanHashes(createdBefore time.Time) ([]string, error) {
	sel := Select("hash").From(r.tableName).
		Where(And{
			Lt{"created_at": createdBefore},
			Expr("hash NOT IN (SELECT hash FROM item_artwork WHERE hash <> '')"),
		})
	var hashes []string
	err := r.queryAllSlice(sel, &hashes)
	return hashes, err
}

func (r *artworkRepository) DeleteImages(hashes ...string) error {
	for chunk := range slices.Chunk(hashes, 200) {
		if err := r.delete(Eq{"hash": chunk}); err != nil {
			return err
		}
	}
	return nil
}

func (r *artworkRepository) GetItemArtwork(kind, id, imageType string) (*model.ItemArtwork, error) {
	sel := Select("*").From("item_artwork").
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
	var res model.ItemArtwork
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *artworkRepository) PutItemArtwork(ia *model.ItemArtwork) error {
	if ia.ImageType == "" {
		ia.ImageType = model.ImageTypePrimary
	}
	ia.UpdatedAt = time.Now()
	ins := Insert("item_artwork").SetMap(map[string]any{
		"item_kind": ia.ItemKind, "item_id": ia.ItemID, "image_type": ia.ImageType,
		"hash": ia.Hash, "source": ia.Source,
		"attempted_at": ia.AttemptedAt, "updated_at": ia.UpdatedAt,
	}).Suffix(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
		hash=excluded.hash, source=excluded.source,
		attempted_at=excluded.attempted_at, updated_at=excluded.updated_at`)
	_, err := r.executeSQL(ins)
	return err
}

func (r *artworkRepository) DeleteForItem(kind, id string) error {
	del := Delete("item_artwork").Where(Eq{"item_kind": kind, "item_id": id})
	_, err := r.executeSQL(del)
	return err
}

func (r *artworkRepository) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	res := map[string]model.ItemArtworkInfo{}
	for chunk := range slices.Chunk(ids, 200) {
		sel := Select("ia.item_id", "ia.hash", "COALESCE(a.blur_hash, '') as blur_hash").
			From("item_artwork ia").
			LeftJoin("artwork a ON a.hash = ia.hash").
			Where(And{
				Eq{"ia.item_kind": kind},
				Eq{"ia.image_type": model.ImageTypePrimary},
				Eq{"ia.item_id": chunk},
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
	}
	return res, nil
}

func (r *artworkRepository) EnqueueStaleAbsent(kind string, attemptedBefore time.Time) (int64, error) {
	now := time.Now()
	ins := Expr(`INSERT INTO artwork_queue (item_kind, item_id, image_type, priority, attempts, retry_at, enqueued_at)
		SELECT item_kind, item_id, image_type, ?, 0, ?, ?
		FROM item_artwork WHERE item_kind = ? AND hash = '' AND attempted_at < ?
		ON CONFLICT (item_kind, item_id, image_type) DO NOTHING`,
		model.ArtworkPriorityRecheck, now, now, kind, attemptedBefore)
	return r.executeSQL(ins)
}

var _ model.ArtworkRepository = (*artworkRepository)(nil)
