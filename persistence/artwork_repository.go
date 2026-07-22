package persistence

import (
	"context"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

const (
	itemArtworkTable = "item_artwork"
	artworkBatchSize = 200
)

type itemArtworkSQL struct {
	sqlRepository
}

type artworkRepository struct {
	sqlRepository
	items itemArtworkSQL
}

func NewArtworkRepository(ctx context.Context, db dbx.Builder) model.ArtworkRepository {
	r := &artworkRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "artwork"
	r.items.ctx = ctx
	r.items.db = db
	r.items.tableName = itemArtworkTable
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
	// created_at is the last-acquisition-write time the prune grace window keys on.
	a.CreatedAt = time.Now()
	values, err := toSQLArgs(*a)
	if err != nil {
		return err
	}
	// created_at=excluded.created_at: reacquiring an orphan must reset the prune grace window.
	ins := Insert(r.tableName).SetMap(values).Suffix(`ON CONFLICT (hash) DO UPDATE SET mime=excluded.mime, width=excluded.width,
		height=excluded.height, size_bytes=excluded.size_bytes, blur_hash=excluded.blur_hash,
		source_path=excluded.source_path, ref_mtime=excluded.ref_mtime, created_at=excluded.created_at`)
	_, err = r.executeSQL(ins)
	return err
}

func (r *artworkRepository) GetImages(hashes []string) (map[string]model.Artwork, error) {
	res := map[string]model.Artwork{}
	for chunk := range slices.Chunk(hashes, artworkBatchSize) {
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

func (r *artworkRepository) GetAllHashes() ([]string, error) {
	sel := Select("hash").From(r.tableName)
	var hashes []string
	err := r.queryAllSlice(sel, &hashes)
	return hashes, err
}

func (r *artworkRepository) GetOrphanHashes(createdBefore time.Time) ([]string, error) {
	sel := Select("hash").From(r.tableName).
		Where(And{
			Lt{"created_at": createdBefore},
			Expr("hash NOT IN (SELECT hash FROM " + itemArtworkTable + " WHERE hash <> '')"),
		})
	var hashes []string
	err := r.queryAllSlice(sel, &hashes)
	return hashes, err
}

func (r *artworkRepository) DeleteImages(hashes ...string) error {
	for chunk := range slices.Chunk(hashes, artworkBatchSize) {
		if err := r.delete(Eq{"hash": chunk}); err != nil {
			return err
		}
	}
	return nil
}

func (r *artworkRepository) GetItemArtwork(kind, id, imageType string) (*model.ItemArtwork, error) {
	sel := Select("*").From(itemArtworkTable).
		Where(Eq{"item_kind": kind, "item_id": id, "image_type": imageType})
	var res model.ItemArtwork
	if err := r.items.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *artworkRepository) PutItemArtwork(ia *model.ItemArtwork) error {
	if ia.ImageType == "" {
		ia.ImageType = model.ImageTypePrimary
	}
	ia.UpdatedAt = time.Now()
	values, err := toSQLArgs(*ia)
	if err != nil {
		return err
	}
	ins := Insert(itemArtworkTable).SetMap(values).Suffix(`ON CONFLICT (item_kind, item_id, image_type) DO UPDATE SET
		hash=excluded.hash, source=excluded.source,
		attempted_at=excluded.attempted_at, updated_at=excluded.updated_at`)
	_, err = r.items.executeSQL(ins)
	return err
}

func (r *artworkRepository) DeleteForItem(kind, id string) error {
	return r.items.delete(Eq{"item_kind": kind, "item_id": id})
}

func (r *artworkRepository) GetInfoForItems(kind string, ids []string) (map[string]model.ItemArtworkInfo, error) {
	res := map[string]model.ItemArtworkInfo{}
	for chunk := range slices.Chunk(ids, artworkBatchSize) {
		sel := Select("ia.item_id", "ia.hash", "COALESCE(a.blur_hash, '') as blur_hash").
			From(itemArtworkTable + " ia").
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
		if err := r.items.queryAll(sel, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			res[row.ItemID] = model.ItemArtworkInfo{
				ItemID: row.ItemID, Hash: row.Hash, BlurHash: row.BlurHash,
			}
		}
	}
	return res, nil
}

var _ model.ArtworkRepository = (*artworkRepository)(nil)
