package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/request"
)

type userPropsRepository struct {
	sqlRepository
}

func NewUserPropsRepository(ctx context.Context, o orm.Ormer) model.UserPropsRepository {
	r := &userPropsRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "user_props"
	return r
}

func (r userPropsRepository) Put(key string, value string) error {
	u, ok := request.UserFrom(r.ctx)
	if !ok {
		return model.ErrInvalidAuth
	}
	update := Update(r.tableName).Set("value", value).Where(And{Eq{"user_id": u.ID}, Eq{"key": key}})
	count, err := r.executeSQL(update)
	if err != nil {
		return nil
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).Columns("user_id", "key", "value").Values(u.ID, key, value)
	_, err = r.executeSQL(insert)
	return err
}

func (r userPropsRepository) Get(key string) (string, error) {
	u, ok := request.UserFrom(r.ctx)
	if !ok {
		return "", model.ErrInvalidAuth
	}
	sel := Select("value").From(r.tableName).Where(And{Eq{"user_id": u.ID}, Eq{"key": key}})
	resp := struct {
		Value string
	}{}
	err := r.queryOne(sel, &resp)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

func (r userPropsRepository) DefaultGet(key string, defaultValue string) (string, error) {
	value, err := r.Get(key)
	if err == model.ErrNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (r userPropsRepository) Delete(key string) error {
	u, ok := request.UserFrom(r.ctx)
	if !ok {
		return model.ErrInvalidAuth
	}
	return r.delete(And{Eq{"user_id": u.ID}, Eq{"key": key}})
}
