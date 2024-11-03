package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type folderRepository struct {
	sqlRepository
}

type dbFolder struct {
	*model.Folder `structs:",flatten"`
	ImageFiles    string `structs:"-" json:"-"`
}

func (f *dbFolder) PostScan() error {
	var err error
	if f.ImageFiles != "" {
		if err = json.Unmarshal([]byte(f.ImageFiles), &f.Folder.ImageFiles); err != nil {
			return fmt.Errorf("parsing folder image files from db: %w", err)
		}
	}
	return nil
}

func (f *dbFolder) PostMapArgs(args map[string]any) error {
	if f.Folder.ImageFiles == nil {
		args["image_files"] = "[]"
	} else {
		imgFiles, err := json.Marshal(f.Folder.ImageFiles)
		if err != nil {
			return fmt.Errorf("marshalling image files: %w", err)
		}
		args["image_files"] = string(imgFiles)
	}
	return nil
}

type dbFolders []dbFolder

func (fs dbFolders) toModels() []model.Folder {
	return slice.Map(fs, func(f dbFolder) model.Folder { return *f.Folder })
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
	var res dbFolder
	err := r.queryOne(sq, res)
	return res.Folder, err
}

func (r folderRepository) GetAll(lib model.Library) ([]model.Folder, error) {
	sq := r.newSelect().Columns("*").Where(Eq{"library_id": lib.ID})
	var res dbFolders
	err := r.queryAll(sq, &res)
	return res.toModels(), err
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

func (r folderRepository) Put(f *model.Folder) error {
	dbf := dbFolder{Folder: f}
	_, err := r.put(dbf.ID, &dbf)
	return err
}

func (r folderRepository) MarkMissing(missing bool, ids ...string) error {
	log.Debug(r.ctx, "Marking folders as missing", "ids", ids, "missing", missing)
	for chunk := range slices.Chunk(ids, 200) {
		sq := Update(r.tableName).
			Set("missing", missing).
			Set("updated_at", timeToSQL(time.Now())).
			Where(Eq{"id": chunk})
		_, err := r.executeSQL(sq)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO Remove?
func (r folderRepository) Touch(lib model.Library, path string, t time.Time) error {
	id := model.FolderID(lib, path)
	sq := Update(r.tableName).Set("updated_at", timeToSQL(t)).Where(Eq{"id": id})
	_, err := r.executeSQL(sq)
	return err
}

var _ model.FolderRepository = (*folderRepository)(nil)
