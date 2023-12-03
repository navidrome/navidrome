package persistence

import (
	"context"
	"errors"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type userPropsRepository struct {
	sqlRepository
}

func NewUserPropsRepository(ctx context.Context, db dbx.Builder) model.UserPropsRepository {
	r := &userPropsRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "user_props"
	return r
}

func (r userPropsRepository) Put(userId, key string, value string) error {
	update := Update(r.tableName).Set("value", value).Where(And{Eq{"user_id": userId}, Eq{"key": key}})
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).Columns("user_id", "key", "value").Values(userId, key, value)
	_, err = r.executeSQL(insert)
	return err
}

func (r userPropsRepository) Get(userId, key string) (string, error) {
	sel := Select("value").From(r.tableName).Where(And{Eq{"user_id": userId}, Eq{"key": key}})
	resp := struct {
		Value string
	}{}
	err := r.queryOne(sel, &resp)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

func (r userPropsRepository) DefaultGet(userId, key string, defaultValue string) (string, error) {
	value, err := r.Get(userId, key)
	if errors.Is(err, model.ErrNotFound) {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (r userPropsRepository) Delete(userId, key string) error {
	return r.delete(And{Eq{"user_id": userId}, Eq{"key": key}})
}
