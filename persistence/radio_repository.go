package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type radioRepository struct {
	sqlRepository
}

func NewRadioRepository(ctx context.Context, db dbx.Builder) model.RadioRepository {
	r := &radioRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Radio{}, map[string]filterFunc{
		"name": containsFilter("name"),
	})
	r.sortMappings = map[string]string{
		"name": "(name collate nocase), name",
	}
	return r
}

func (r *radioRepository) isPermitted() bool {
	user := loggedUser(r.ctx)
	return user.IsAdmin
}

func (r *radioRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelect()
	return r.count(sql, options...)
}

func (r *radioRepository) Delete(id string) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	return r.delete(Eq{"id": id})
}

func (r *radioRepository) Get(id string) (*model.Radio, error) {
	sel := r.newSelect().Where(Eq{"id": id}).Columns("*")
	res := model.Radio{}
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *radioRepository) GetAll(options ...model.QueryOptions) (model.Radios, error) {
	sel := r.newSelect(options...).Columns("*")
	res := model.Radios{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *radioRepository) Put(radio *model.Radio) error {
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}

	var values map[string]interface{}

	radio.UpdatedAt = time.Now()

	if radio.ID == "" {
		radio.CreatedAt = time.Now()
		radio.ID = strings.ReplaceAll(uuid.NewString(), "-", "")
		values, _ = toSQLArgs(*radio)
	} else {
		values, _ = toSQLArgs(*radio)
		update := Update(r.tableName).Where(Eq{"id": radio.ID}).SetMap(values)
		count, err := r.executeSQL(update)

		if err != nil {
			return err
		} else if count > 0 {
			return nil
		}
	}

	values["created_at"] = time.Now()
	insert := Insert(r.tableName).SetMap(values)
	_, err := r.executeSQL(insert)
	return err
}

func (r *radioRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(r.ctx, options...))
}

func (r *radioRepository) EntityName() string {
	return "radio"
}

func (r *radioRepository) NewInstance() interface{} {
	return &model.Radio{}
}

func (r *radioRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *radioRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *radioRepository) Save(entity interface{}) (string, error) {
	t := entity.(*model.Radio)
	if !r.isPermitted() {
		return "", rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return "", rest.ErrNotFound
	}
	return t.ID, err
}

func (r *radioRepository) Update(id string, entity interface{}, cols ...string) error {
	t := entity.(*model.Radio)
	t.ID = id
	if !r.isPermitted() {
		return rest.ErrPermissionDenied
	}
	err := r.Put(t)
	if errors.Is(err, model.ErrNotFound) {
		return rest.ErrNotFound
	}
	return err
}

var _ model.RadioRepository = (*radioRepository)(nil)
var _ rest.Repository = (*radioRepository)(nil)
var _ rest.Persistable = (*radioRepository)(nil)
