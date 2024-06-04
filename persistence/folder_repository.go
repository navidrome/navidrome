package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type folderRepository struct {
	sqlRepository
}

func newFolderRepository(ctx context.Context, db dbx.Builder) model.FolderRepository {
	r := &folderRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "folder"
	return r
}

func (r folderRepository) Get(lib model.Library, path string) (*model.Folder, error) {
	id := model.NewFolder(lib, path).ID
	sq := r.newSelect().Where(Eq{"id": id})
	var res model.Folder
	err := r.queryOne(sq, res)
	return &res, err
}

func (r folderRepository) GetAll(lib model.Library) ([]model.Folder, error) {
	sq := r.newSelect().Columns("*").Where(Eq{"library_id": lib.ID})
	var res []model.Folder
	err := r.queryAll(sq, &res)
	return res, err
}

func (r folderRepository) GetLastUpdates(lib model.Library) (map[string]time.Time, error) {
	sq := r.newSelect().Columns("id", "updated_at").Where(Eq{"library_id": lib.ID, "missing": false})
	var res []struct {
		ID        string
		UpdatedAt time.Time
	}
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	m := make(map[string]time.Time, len(res))
	for _, f := range res {
		m[f.ID] = f.UpdatedAt
	}
	return m, nil
}

func (r folderRepository) Put(lib model.Library, path string) error {
	folder := model.NewFolder(lib, path)
	folder.Missing = false
	_, err := r.put(folder.ID, folder)
	return err
}

func (r folderRepository) MarkMissing(missing bool, ids ...string) error {
	log.Debug(r.ctx, "Marking folders as missing", "ids", ids, "missing", missing)
	sq := Update(r.tableName).
		Set("missing", missing).
		Set("updated_at", timeToSQL(time.Now())).
		Where(Eq{"id": ids})
	_, err := r.executeSQL(sq)
	return err
}

// TODO Remove?
func (r folderRepository) Touch(lib model.Library, path string, t time.Time) error {
	id := model.FolderID(lib, path)
	sq := Update(r.tableName).Set("updated_at", timeToSQL(t)).Where(Eq{"id": id})
	_, err := r.executeSQL(sq)
	return err
}

var _ model.FolderRepository = (*folderRepository)(nil)
