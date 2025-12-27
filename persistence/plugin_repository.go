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

	// Try update first
	values, _ := toSQLArgs(*plugin)
	update := Update(r.tableName).Where(Eq{"id": plugin.ID}).SetMap(values)
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	// Insert if not exists
	plugin.CreatedAt = time.Now()
	values, _ = toSQLArgs(*plugin)
	insert := Insert(r.tableName).SetMap(values)
	_, err = r.executeSQL(insert)
	return err
}

func (r *pluginRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *pluginRepository) EntityName() string {
	return "plugin"
}

func (r *pluginRepository) NewInstance() interface{} {
	return &model.Plugin{}
}

func (r *pluginRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *pluginRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *pluginRepository) Save(entity interface{}) (string, error) {
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

func (r *pluginRepository) Update(id string, entity interface{}, cols ...string) error {
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
