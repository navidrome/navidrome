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

func NewPluginRepository(db dbx.Builder) model.PluginRepository {
	r := &pluginRepository{}
	r.db = db
	r.registerModel(&model.Plugin{}, map[string]filterFunc{
		"id":      idFilter("plugin"),
		"enabled": booleanFilter,
	})
	return r
}

func (r *pluginRepository) isPermitted(ctx context.Context) bool {
	user := loggedUser(ctx)
	return user.IsAdmin
}

func (r *pluginRepository) ClearErrors(ctx context.Context) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	_, err := r.db.NewQuery("UPDATE plugin SET last_error = '' WHERE last_error != ''").Execute()
	return err
}

func (r *pluginRepository) CountAll(ctx context.Context, options ...model.QueryOptions) (int64, error) {
	if !r.isPermitted(ctx) {
		return 0, rest.ErrPermissionDenied
	}
	sql := r.newSelect(ctx)
	return r.count(ctx, sql, options...)
}

func (r *pluginRepository) Delete(ctx context.Context, id string) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}
	return r.delete(ctx, Eq{"id": id})
}

func (r *pluginRepository) Get(ctx context.Context, id string) (*model.Plugin, error) {
	if !r.isPermitted(ctx) {
		return nil, rest.ErrPermissionDenied
	}
	sel := r.newSelect(ctx).Where(Eq{"id": id}).Columns("*")
	res := model.Plugin{}
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

func (r *pluginRepository) GetAll(ctx context.Context, options ...model.QueryOptions) (model.Plugins, error) {
	if !r.isPermitted(ctx) {
		return nil, rest.ErrPermissionDenied
	}
	sel := r.newSelect(ctx, options...).Columns("*")
	res := model.Plugins{}
	err := r.queryAll(ctx, sel, &res)
	return res, err
}

func (r *pluginRepository) Put(ctx context.Context, plugin *model.Plugin) error {
	if !r.isPermitted(ctx) {
		return rest.ErrPermissionDenied
	}

	plugin.UpdatedAt = time.Now()

	if plugin.ID == "" {
		return errors.New("plugin ID cannot be empty")
	}

	// Upsert using INSERT ... ON CONFLICT for atomic operation
	_, err := r.db.NewQuery(`
		INSERT INTO plugin (id, path, manifest, config, users, all_users, libraries, all_libraries, allow_write_access, enabled, last_error, sha256, created_at, updated_at)
		VALUES ({:id}, {:path}, {:manifest}, {:config}, {:users}, {:all_users}, {:libraries}, {:all_libraries}, {:allow_write_access}, {:enabled}, {:last_error}, {:sha256}, {:created_at}, {:updated_at})
		ON CONFLICT(id) DO UPDATE SET
			path = excluded.path,
			manifest = excluded.manifest,
			config = excluded.config,
			users = excluded.users,
			all_users = excluded.all_users,
			libraries = excluded.libraries,
			all_libraries = excluded.all_libraries,
			allow_write_access = excluded.allow_write_access,
			enabled = excluded.enabled,
			last_error = excluded.last_error,
			sha256 = excluded.sha256,
			updated_at = excluded.updated_at
	`).Bind(dbx.Params{
		"id":                 plugin.ID,
		"path":               plugin.Path,
		"manifest":           plugin.Manifest,
		"config":             plugin.Config,
		"users":              plugin.Users,
		"all_users":          plugin.AllUsers,
		"libraries":          plugin.Libraries,
		"all_libraries":      plugin.AllLibraries,
		"allow_write_access": plugin.AllowWriteAccess,
		"enabled":            plugin.Enabled,
		"last_error":         plugin.LastError,
		"sha256":             plugin.SHA256,
		"created_at":         time.Now(),
		"updated_at":         plugin.UpdatedAt,
	}).Execute()
	return err
}

func (r *pluginRepository) Count(ctx context.Context, options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(ctx, r.parseRestOptions(ctx, options...))
}

func (r *pluginRepository) Read(ctx context.Context, id string) (*model.Plugin, error) {
	return r.Get(ctx, id)
}

func (r *pluginRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Plugin, error) {
	return r.GetAll(ctx, r.parseRestOptions(ctx, options...))
}

var _ model.PluginRepository = (*pluginRepository)(nil)
var _ rest.Repository[model.Plugin] = (*pluginRepository)(nil)
