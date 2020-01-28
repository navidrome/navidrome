package persistence

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/model"
)

type property struct {
	ID    string `orm:"pk;column(id)"`
	Value string
}

type propertyRepository struct {
	sqlRepository
}

func NewPropertyRepository(ctx context.Context, o orm.Ormer) model.PropertyRepository {
	r := &propertyRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "property"
	return r
}

func (r propertyRepository) Put(id string, value string) error {
	update := squirrel.Update(r.tableName).Set("value", value).Where(squirrel.Eq{"id": id})
	count, err := r.executeSQL(update)
	if err != nil {
		return nil
	}
	if count > 0 {
		return nil
	}
	insert := squirrel.Insert(r.tableName).Columns("id", "value").Values(id, value)
	_, err = r.executeSQL(insert)
	return err
}

func (r propertyRepository) Get(id string) (string, error) {
	sel := squirrel.Select("value").From(r.tableName).Where(squirrel.Eq{"id": id})
	resp := struct {
		Value string
	}{}
	err := r.queryOne(sel, &resp)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

func (r propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	value, err := r.Get(id)
	if err == model.ErrNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}
