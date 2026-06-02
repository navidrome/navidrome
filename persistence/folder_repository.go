package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
	"github.com/pocketbase/dbx"
)

type folderRepository struct {
	sqlRepository
}

type dbFolder struct {
	model.Folder    `structs:",flatten"`
	DBImageFiles    string    `db:"image_files" structs:"image_files"`
	ImagesUpdatedAt time.Time `db:"images_updated_at" structs:"images_updated_at"`
	LibraryPath     string    `structs:"-" db:"library_path"`
	LibraryName     string    `structs:"-" db:"library_name"`
}

func (f *dbFolder) PostScan() error {
	var err error
	f.Folder.ImagesUpdatedAt = f.ImagesUpdatedAt
	if f.DBImageFiles != "" {

		if err = json.Unmarshal([]byte(f.DBImageFiles), &f.Folder.ImageFiles); err != nil {
			return fmt.Errorf("parsing folder image files from db: %w", err)
		}
	}
	return nil
}

func (f *dbFolder) PostMapArgs(args map[string]any) error {
	imgFiles := "[]"
	if f.Folder.ImageFiles != nil {
		b, err := json.Marshal(f.Folder.ImageFiles)
		if err != nil {
			return fmt.Errorf("marshalling image files: %w", err)
		}
		imgFiles = string(b)
	}
	args["image_files"] = imgFiles
	args["images_updated_at"] = f.Folder.ImagesUpdatedAt
	return nil
}

type dbFolders []dbFolder

func (fs dbFolders) toModels() model.Folders {
	return slice.Map(fs, func(f dbFolder) model.Folder { return f.Folder })
}

func newFolderRepository(ctx context.Context, db dbx.Builder) model.FolderRepository {
	r := &folderRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "folder"
	r.registerModel(&model.Folder{}, map[string]filterFunc{
		"id":         idFilter("folder"),
		"parent_id":  eqFilter,
		"library_id": libraryIdFilter,
		"missing":    booleanFilter,
	})
	r.setSortMappings(map[string]string{
		"name": "folder.name",
		"path": "folder.path",
	})
	return r
}

func (r *folderRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *folderRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *folderRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *folderRepository) EntityName() string {
	return "folder"
}

func (r *folderRepository) NewInstance() any {
	return &model.Folder{}
}

func (r folderRepository) selectFolder(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelect(options...).Columns("folder.*", "library.path as library_path", "library.name as library_name").
		Join("library on library.id = folder.library_id")
	return r.applyLibraryFilter(sql)
}

func (r folderRepository) populateBreadcrumbs(f *model.Folder, libPath, libName string) {
	lib := model.Library{ID: f.LibraryID, Path: libPath, Name: libName}
	libIDStr := strconv.Itoa(lib.ID)
	f.Breadcrumbs = []model.Breadcrumb{
		{ID: libIDStr, Name: lib.Name},
	}

	if f.Path == "" || f.Path == "." {
		return
	}

	parts := strings.Split(strings.Trim(f.Path, "/"), "/")
	currentPath := ""
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if currentPath == "" {
			currentPath = part
		} else {
			currentPath = currentPath + "/" + part
		}
		f.Breadcrumbs = append(f.Breadcrumbs, model.Breadcrumb{
			ID:   model.FolderID(lib, currentPath),
			Name: part,
		})
	}
}

func (r folderRepository) Get(id string) (*model.Folder, error) {
	sq := r.selectFolder().Where(Eq{"folder.id": id})
	var res dbFolder
	err := r.queryOne(sq, &res)
	if err == nil {
		r.populateBreadcrumbs(&res.Folder, res.LibraryPath, res.LibraryName)
	}
	return &res.Folder, err
}

func (r folderRepository) GetByPath(lib model.Library, path string) (*model.Folder, error) {
	id := model.NewFolder(lib, path).ID
	return r.Get(id)
}

func (r folderRepository) GetAll(opt ...model.QueryOptions) (model.Folders, error) {
	sq := r.selectFolder(opt...)
	var res dbFolders
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	for i := range res {
		r.populateBreadcrumbs(&res[i].Folder, res[i].LibraryPath, res[i].LibraryName)
	}
	return res.toModels(), nil
}

func (r folderRepository) CountAll(opt ...model.QueryOptions) (int64, error) {
	query := r.newSelect()
	query = r.applyLibraryFilter(query)
	return r.count(query, opt...)
}

func (r folderRepository) GetFolderUpdateInfo(lib model.Library, targetPaths ...string) (map[string]model.FolderUpdateInfo, error) {
	if len(targetPaths) == 0 {
		return r.getFolderUpdateInfoAll(lib)
	}
	for _, targetPath := range targetPaths {
		if targetPath == "" || targetPath == "." {
			return r.getFolderUpdateInfoAll(lib)
		}
	}
	const batchSize = 100
	result := make(map[string]model.FolderUpdateInfo)
	for batch := range slices.Chunk(targetPaths, batchSize) {
		batchResult, err := r.getFolderUpdateInfoBatch(lib, batch)
		if err != nil {
			return nil, err
		}
		maps.Copy(result, batchResult)
	}
	return result, nil
}

func (r folderRepository) getFolderUpdateInfoAll(lib model.Library) (map[string]model.FolderUpdateInfo, error) {
	where := And{
		Eq{"library_id": lib.ID},
		Eq{"missing": false},
	}
	return r.queryFolderUpdateInfo(where)
}

func (r folderRepository) getFolderUpdateInfoBatch(lib model.Library, targetPaths []string) (map[string]model.FolderUpdateInfo, error) {
	where := And{
		Eq{"library_id": lib.ID},
		Eq{"missing": false},
	}
	folderIDs := make([]string, 0, len(targetPaths))
	pathConditions := make(Or, 0, len(targetPaths)*2)
	for _, targetPath := range targetPaths {
		cleanPath := strings.TrimPrefix(targetPath, string(os.PathSeparator))
		cleanPath = filepath.Clean(cleanPath)
		folderIDs = append(folderIDs, model.FolderID(lib, cleanPath))
		pathConditions = append(pathConditions, Eq{"path": cleanPath})
		pathConditions = append(pathConditions, Like{"path": cleanPath + "/%"})
	}
	if len(folderIDs) > 0 {
		where = append(where, Or{Eq{"id": folderIDs}, pathConditions})
	} else if len(pathConditions) > 0 {
		where = append(where, pathConditions)
	}
	return r.queryFolderUpdateInfo(where)
}

func (r folderRepository) queryFolderUpdateInfo(where And) (map[string]model.FolderUpdateInfo, error) {
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
	dbf := dbFolder{Folder: *f}
	_, err := r.put(dbf.ID, &dbf)
	return err
}

func (r folderRepository) MarkMissing(missing bool, ids ...string) error {
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
	return wrapFolderCursor(cursor), nil
}

func wrapFolderCursor(cursor iter.Seq2[dbFolder, error]) model.FolderCursor {
	return func(yield func(model.Folder, error) bool) {
		for f, err := range cursor {
			if !yield(f.Folder, err) || err != nil {
				return
			}
		}
	}
}

func (r folderRepository) purgeEmpty(libraryIDs ...int) error {
	sq := Delete(r.tableName).Where(And{
		Eq{"num_audio_files": 0},
		Eq{"num_playlists": 0},
		Eq{"image_files": "[]"},
		ConcatExpr("id not in (select parent_id from folder)"),
		ConcatExpr("id not in (select folder_id from media_file)"),
	})
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
