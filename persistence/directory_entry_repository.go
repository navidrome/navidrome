package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type directoryEntryRepository struct {
	sqlRepository
	ctx       context.Context
	hardcoded map[string]model.DirectoryEntry
}

func NewDirectoryEntryRepository(ctx context.Context, o orm.QueryExecutor) model.DirectoryEntryRepository {
	d := &directoryEntryRepository{}
	d.ctx = ctx
	d.ormer = o
	d.tableName = "directory_entry"

	d.hardcoded = map[string]model.DirectoryEntry{
		conf.Server.MusicFolderId: {
			ID:   conf.Server.MusicFolderId,
			Path: conf.Server.MusicFolder,
			Name: "Music Library",
		},
	}
	return d
}

func (d *directoryEntryRepository) BrowserDirectory(id string) (model.DirectoryEntiesOrFiles, error) {
	if id == "" {
		id = conf.Server.MusicFolderId
	}

	sel := d.newSelect().
		Columns("de.id folder_id, de.name folder_name", "de.path != '' is_dir", "de.parent_id parent_id", "mf.*").
		Column("de.id = ? is_parent", id).
		From("directory_entry de").
		LeftJoin("media_file mf ON mf.id = de.id").
		Where(Or{
			Eq{"parent_id": id},
			And{Eq{"de.id": id}, Eq{"is_dir": true}},
		}).
		OrderBy("is_parent desc")
	res := model.DirectoryEntiesOrFiles{}
	err := d.queryAll(sel, &res)
	return res, err
}

func (d *directoryEntryRepository) Delete(id string) error {
	err := d.delete(Eq{"id": id})
	return err
}

func (d *directoryEntryRepository) Get(id string) (*model.DirectoryEntry, error) {
	if id == "" {
		id = conf.Server.MusicFolderId
	}
	sel := d.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.DirectoryEntry{}
	err := d.queryOne(sel, &res)

	if err != nil {
		if folder, ok := d.hardcoded[id]; ok {
			log.Error(d.ctx, "Error getting default media folder. Returning hardcoded", "err", err)
			return &folder, nil
		}
	}

	return &res, err
}

func (d *directoryEntryRepository) GetDbRoot() (model.DirectoryEntries, error) {
	sel := d.newSelect().Where(Eq{"parent_id": nil}).Columns("*")
	res := model.DirectoryEntries{}
	err := d.queryAll(sel, &res)
	return res, err
}

func (d *directoryEntryRepository) GetAllDirectories() (model.DirectoryEntries, error) {
	sel := d.newSelect().Columns("*").OrderBy("path")
	res := model.DirectoryEntries{}
	err := d.queryAll(sel, &res)
	return res, err
}

func (d *directoryEntryRepository) Put(folder *model.DirectoryEntry) error {
	final_values := map[string]interface{}{}
	values, _ := toSqlArgs(folder)

	for k, v := range values {
		if v != "" {
			final_values[k] = v
		}
	}

	query := Insert(d.tableName).Options("OR IGNORE").SetMap(final_values)
	_, err := d.executeSQL(query)
	return err
}

var _ model.DirectoryEntryRepository = (*directoryEntryRepository)(nil)
