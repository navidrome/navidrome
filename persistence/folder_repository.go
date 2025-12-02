package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func (r folderRepository) selectFolder(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("folder.*", "library.path as library_path").
		Join("library on library.id = folder.library_id")
	return r.applyLibraryFilter(sql)
}

func (r folderRepository) Get(id string) (*model.Folder, error) {
	sq := r.selectFolder().Where(Eq{"folder.id": id})
	var res dbFolder
	err := r.queryOne(sq, &res)
	return res.Folder, err
}

func (r folderRepository) GetByPath(lib model.Library, path string) (*model.Folder, error) {
	id := model.NewFolder(lib, path).ID
	return r.Get(id)
}

func (r folderRepository) GetAll(opt ...model.QueryOptions) ([]model.Folder, error) {
	sq := r.selectFolder(opt...)
	var res dbFolders
	err := r.queryAll(sq, &res)
	return res.toModels(), err
}

func (r folderRepository) CountAll(opt ...model.QueryOptions) (int64, error) {
	query := r.newSelect(opt...).Columns("count(*)")
	query = r.applyLibraryFilter(query)
	return r.count(query)
}

func (r folderRepository) GetFolderUpdateInfo(lib model.Library, targetPaths ...string) (map[string]model.FolderUpdateInfo, error) {
	where := And{
		Eq{"library_id": lib.ID},
		Eq{"missing": false},
	}

	// If specific paths are requested, include those folders and all their descendants
	if len(targetPaths) > 0 {
		// Collect folder IDs for exact target folders and path conditions for descendants
		folderIDs := make([]string, 0, len(targetPaths))
		pathConditions := make(Or, 0, len(targetPaths)*2)

		for _, targetPath := range targetPaths {
			if targetPath == "" || targetPath == "." {
				// Root path - include everything in this library
				pathConditions = Or{}
				folderIDs = nil
				break
			}
			// Clean the path to normalize it. Paths stored in the folder table do not have leading/trailing slashes.
			cleanPath := strings.TrimPrefix(targetPath, string(os.PathSeparator))
			cleanPath = filepath.Clean(cleanPath)

			// Include the target folder itself by ID
			folderIDs = append(folderIDs, model.FolderID(lib, cleanPath))

			// Include all descendants: folders whose path field equals or starts with the target path
			// Note: Folder.Path is the directory path, so children have path = targetPath
			pathConditions = append(pathConditions, Eq{"path": cleanPath})
			pathConditions = append(pathConditions, Like{"path": cleanPath + "/%"})
		}

		// Combine conditions: exact folder IDs OR descendant path patterns
		if len(folderIDs) > 0 {
			where = append(where, Or{Eq{"id": folderIDs}, pathConditions})
		} else if len(pathConditions) > 0 {
			where = append(where, pathConditions)
		}
	}

	sq := r.newSelect().Columns("id", "updated_at", "hash").Where(where)
	var res []struct {
		ID        string
		UpdatedAt time.Time
		Hash      string
	}
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	m := make(map[string]model.FolderUpdateInfo, len(res))
	for _, f := range res {
		m[f.ID] = model.FolderUpdateInfo{UpdatedAt: f.UpdatedAt, Hash: f.Hash}
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
			Set("updated_at", time.Now()).
			Where(Eq{"id": chunk})
		_, err := r.executeSQL(sq)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r folderRepository) GetTouchedWithPlaylists() (model.FolderCursor, error) {
	query := r.selectFolder().Where(And{
		Eq{"missing": false},
		Gt{"num_playlists": 0},
		ConcatExpr("folder.updated_at > library.last_scan_at"),
	})
	cursor, err := queryWithStableResults[dbFolder](r.sqlRepository, query)
	if err != nil {
		return nil, err
	}
	return func(yield func(model.Folder, error) bool) {
		for f, err := range cursor {
			if !yield(*f.Folder, err) || err != nil {
				return
			}
		}
	}, nil
}

func (r folderRepository) purgeEmpty(libraryIDs ...int) error {
	sq := Delete(r.tableName).Where(And{
		Eq{"num_audio_files": 0},
		Eq{"num_playlists": 0},
		Eq{"image_files": "[]"},
		ConcatExpr("id not in (select parent_id from folder)"),
		ConcatExpr("id not in (select folder_id from media_file)"),
	})
	// If libraryIDs are specified, only purge folders from those libraries
	if len(libraryIDs) > 0 {
		sq = sq.Where(Eq{"library_id": libraryIDs})
	}
	c, err := r.executeSQL(sq)
	if err != nil {
		return fmt.Errorf("purging empty folders: %w", err)
	}
	if c > 0 {
		log.Debug(r.ctx, "Purging empty folders", "totalDeleted", c)
	}
	return nil
}

var _ model.FolderRepository = (*folderRepository)(nil)
