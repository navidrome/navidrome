package persistence

import (
	"context"
	"errors"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type pluginRepository struct {
	sqlRepository
}

func NewPluginRepository(ctx context.Context, db dbx.Builder) model.PluginRepository {
	r := &pluginRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Plugin{}, map[string]filterFunc{
		"id":      idFilter("plugin"),
		"enabled": booleanFilter,
	})
	return r
}

func (r *pluginRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *pluginRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	if !r.isPermitted() {
		return 0, rest.ErrPermissionDenied
	}
	sql := r.newSelect()
	return r.count(sql, options...)
}

func (r *pluginRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	return r.delete(Eq{"id": id})
}

func (r *pluginRepository) Get(id string) (*model.Plugin, error) {
	if !r.isPermitted() {
		return nil, rest.ErrPermissionDenied
	}
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.Plugin{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *pluginRepository) GetAll(options ...model.QueryOptions) (model.Plugins, error) {
	if !r.isPermitted() {
		return nil, rest.ErrPermissionDenied
	}
	sel := r.newSelect(options...).Columns("*")
	res := model.Plugins{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *pluginRepository) Put(plugin *model.Plugin) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	plugin.UpdatedAt = time.Now()

	if plugin.ID == "" {
		return errors.New("plugin ID cannot be empty")
	}

	// Upsert using INSERT ... ON CONFLICT for atomic operation
	_, err := r.db.NewQuery(`
		INSERT INTO plugin (id, path, manifest, config, users, all_users, libraries, all_libraries, enabled, last_error, sha256, created_at, updated_at)
		VALUES ({:id}, {:path}, {:manifest}, {:config}, {:users}, {:all_users}, {:libraries}, {:all_libraries}, {:enabled}, {:last_error}, {:sha256}, {:created_at}, {:updated_at})
		ON CONFLICT(id) DO UPDATE SET
			path = excluded.path,
			manifest = excluded.manifest,
			config = excluded.config,
			users = excluded.users,
			all_users = excluded.all_users,
			libraries = excluded.libraries,
			all_libraries = excluded.all_libraries,
			enabled = excluded.enabled,
			last_error = excluded.last_error,
			sha256 = excluded.sha256,
			updated_at = excluded.updated_at
	`).Bind(dbx.Params{
		"id":            plugin.ID,
		"path":          plugin.Path,
		"manifest":      plugin.Manifest,
		"config":        plugin.Config,
		"users":         plugin.Users,
		"all_users":     plugin.AllUsers,
		"libraries":     plugin.Libraries,
		"all_libraries": plugin.AllLibraries,
		"enabled":       plugin.Enabled,
		"last_error":    plugin.LastError,
		"sha256":        plugin.SHA256,
		"created_at":    time.Now(),
		"updated_at":    plugin.UpdatedAt,
	}).Execute()
	return err
}

func (r *pluginRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *pluginRepository) EntityName() string {
	return "plugin"
}

func (r *pluginRepository) NewInstance() any {
	return &model.Plugin{}
}

func (r *pluginRepository) Read(id string) (any, error) {
	return r.Get(id)
}

func (r *pluginRepository) ReadAll(options ...rest.QueryOptions) (any, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *pluginRepository) Save(entity any) (string, error) {
	p := entity.(*model.Plugin)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(p)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return p.ID, err
}

func (r *pluginRepository) Update(id string, entity any, cols ...string) error {
	p := entity.(*model.Plugin)
	p.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(p)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.PluginRepository = (*pluginRepository)(nil)
var _ rest.Repository = (*pluginRepository)(nil)
var _ rest.Persistable = (*pluginRepository)(nil)
