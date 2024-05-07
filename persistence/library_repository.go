package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type libraryRepository struct {
	sqlRepository
	sqlRestful
}

func NewLibraryRepository(ctx context.Context, db dbx.Builder) model.LibraryRepository {
	r := &libraryRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "library"
	return r
}

func (r *libraryRepository) Get(id int) (*model.Library, error) {
	sq := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Library
	err := r.queryOne(sq, &res)
	return &res, err
}

func (r *libraryRepository) Put(l *model.Library) error {
	cols := map[string]any{
		"name":         l.Name,
		"path":         l.Path,
		"remote_path":  l.RemotePath,
		"last_scan_at": l.LastScanAt,
		"updated_at":   time.Now(),
	}
	if l.ID != 0 {
		cols["id"] = l.ID
	}

	sq := Insert(r.tableName).SetMap(cols).
		Suffix(`ON CONFLICT(id) DO UPDATE set name = excluded.name, path = excluded.path, 
					remote_path = excluded.remote_path, last_scan_at = excluded.last_scan_at`)
	_, err := r.executeSQL(sq)
	return err
}

func (r *libraryRepository) GetAll(ops ...model.QueryOptions) (model.Libraries, error) {
	sq := r.newSelect(ops...).Columns("*")
	res := model.Libraries{}
	err := r.queryAll(sq, &res)
	return res, err
}

var _ model.LibraryRepository = (*libraryRepository)(nil)
