package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type mediaFolderRepository struct {
	sqlRepository
	ctx       context.Context
	hardcoded map[string]model.MediaFolder
}

func NewMediaFolderRepository(ctx context.Context, o orm.QueryExecutor) model.MediaFolderRepository {
	r := &mediaFolderRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "media_folder"

	r.hardcoded = map[string]model.MediaFolder{
		conf.Server.MusicFolderId: {
			ID:   conf.Server.MusicFolderId,
			Path: conf.Server.MusicFolder,
			Name: "Music Library",
		},
	}
	return r
}

func (r *mediaFolderRepository) BrowserDirectory(id string) (model.MediaFolderOrFiles, error) {
	if id == "" {
		id = conf.Server.MusicFolderId
	}

	sel := r.newSelect().
		Columns("mo.id folder_id, mo.name folder_name", "mo.path != '' is_dir", "mo.parent_id parent_id", "mf.*").
		Column("mo.id = ? is_parent", id).
		From("media_folder mo").
		LeftJoin("media_file mf ON mf.id = mo.id").
		Where(Or{
			Eq{"parent_id": id},
			And{Eq{"mo.id": id}, Eq{"is_dir": true}},
		}).
		OrderBy("is_parent desc")
	res := model.MediaFolderOrFiles{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *mediaFolderRepository) Delete(id string) error {
	err := r.delete(Eq{"id": id})
	return err
}

func (r *mediaFolderRepository) Get(id string) (*model.MediaFolder, error) {
	if id == "" {
		id = conf.Server.MusicFolderId
	}
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.MediaFolder{}
	err := r.queryOne(sel, &res)

	if err != nil {
		if folder, ok := r.hardcoded[id]; ok {
			log.Error(r.ctx, "Error getting default media folder. Returning hardcoded", "err", err)
			return &folder, nil
		}
	}

	return &res, err
}

func (r *mediaFolderRepository) GetDbRoot() (model.MediaFolders, error) {
	sel := r.newSelect().Where(Eq{"parent_id": nil}).Columns("*")
	res := model.MediaFolders{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *mediaFolderRepository) GetRoot() (model.MediaFolders, error) {
	res := model.MediaFolders{r.hardcoded[conf.Server.MusicFolderId]}
	return res, nil
}

func (r *mediaFolderRepository) GetAllDirectories() (model.MediaFolders, error) {
	sel := r.newSelect().Columns("*").OrderBy("path")
	res := model.MediaFolders{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *mediaFolderRepository) Put(folder *model.MediaFolder) error {
	final_values := map[string]interface{}{}
	values, _ := toSqlArgs(folder)

	for k, v := range values {
		if v != "" {
			final_values[k] = v
		}
	}

	query := Insert(r.tableName).Options("OR IGNORE").SetMap(final_values)
	_, err := r.executeSQL(query)
	return err
}

var _ model.MediaFolderRepository = (*mediaFolderRepository)(nil)
