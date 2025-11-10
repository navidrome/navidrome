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

func (r folderRepository) GetByPaths(targets []model.LibraryPath) (map[string]model.FolderUpdateInfo, error) {
	if len(targets) == 0 {
		return make(map[string]model.FolderUpdateInfo), nil
	}

	// Group targets by library to build efficient queries
	targetsByLib := make(map[int][]string)
	folderIDs := make([]string, 0, len(targets))

	// We need to resolve library paths to generate folder IDs
	// Get all libraries first
	libRepo := NewLibraryRepository(r.ctx, r.db)
	allLibs, err := libRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("getting libraries: %w", err)
	}
	libMap := make(map[int]model.Library)
	for _, lib := range allLibs {
		libMap[lib.ID] = lib
	}

	// Generate folder IDs for all targets
	for _, target := range targets {
		lib, ok := libMap[target.LibraryID]
		if !ok {
			continue // Skip invalid library IDs
		}
		folderPath := target.FolderPath
		if folderPath == "" {
			folderPath = "."
		}
		folderID := model.FolderID(lib, folderPath)
		folderIDs = append(folderIDs, folderID)
		targetsByLib[target.LibraryID] = append(targetsByLib[target.LibraryID], folderPath)
	}

	if len(folderIDs) == 0 {
		return make(map[string]model.FolderUpdateInfo), nil
	}

	// Query folders by IDs
	sq := r.newSelect().Columns("id", "updated_at", "hash").Where(And{
		Eq{"id": folderIDs},
		Eq{"missing": false},
	})
	var res []struct {
		ID        string
		UpdatedAt time.Time
		Hash      string
	}
	err = r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	m := make(map[string]model.FolderUpdateInfo, len(res))
	for _, f := range res {
		m[f.ID] = model.FolderUpdateInfo{UpdatedAt: f.UpdatedAt, Hash: f.Hash}
	}
	return m, nil
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

func (r folderRepository) GetLastUpdates(lib model.Library) (map[string]model.FolderUpdateInfo, error) {
	sq := r.newSelect().Columns("id", "updated_at", "hash").Where(Eq{"library_id": lib.ID, "missing": false})
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

func (r folderRepository) purgeEmpty() error {
	sq := Delete(r.tableName).Where(And{
		Eq{"num_audio_files": 0},
		Eq{"num_playlists": 0},
		Eq{"image_files": "[]"},
		ConcatExpr("id not in (select parent_id from folder)"),
		ConcatExpr("id not in (select folder_id from media_file)"),
	})
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
