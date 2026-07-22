package persistence

import (
	"context"
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

func (r *artworkRepository) Get(hash string) (*model.Artwork, error) {
	sel := Select("*").From(r.tableName).Where(Eq{"hash": hash})
	var res model.Artwork
	if err := r.queryOne(sel, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *artworkRepository) Put(a *model.Artwork) error {
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

func (r *artworkRepository) GetBatch(hashes []string) (map[string]model.Artwork, error) {
	res := map[string]model.Artwork{}
	if len(hashes) == 0 {
		return res, nil
	}
	sel := Select("*").From(r.tableName).Where(Eq{"hash": hashes})
	var all []model.Artwork
	if err := r.queryAll(sel, &all); err != nil {
		return nil, err
	}
	for _, a := range all {
		res[a.Hash] = a
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

func (r *artworkRepository) Delete(hashes ...string) error {
	if len(hashes) == 0 {
		return nil
	}
	return r.delete(Eq{"hash": hashes})
}

var _ model.ArtworkRepository = (*artworkRepository)(nil)
