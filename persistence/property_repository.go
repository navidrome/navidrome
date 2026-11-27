package persistence

import (
	"context"
	"errors"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type propertyRepository struct {
	sqlRepository
}

func NewPropertyRepository(db dbx.Builder) model.PropertyRepository {
	r := &propertyRepository{}
	r.db = db
	r.tableName = "property"
	return r
}

func (r propertyRepository) Put(ctx context.Context, id string, value string) error {
	update := Update(r.tableName).Set("value", value).Where(Eq{"id": id})
	count, err := r.executeSQL(ctx, update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).Columns("id", "value").Values(id, value)
	_, err = r.executeSQL(ctx, insert)
	return err
}

func (r propertyRepository) Get(ctx context.Context, id string) (string, error) {
	sel := Select("value").From(r.tableName).Where(Eq{"id": id})
	resp := struct {
		Value string
	}{}
	err := r.queryOne(ctx, sel, &resp)
	if err != nil {
		return "", err
	}
	return resp.Value, nil
}

func (r propertyRepository) DefaultGet(ctx context.Context, id string, defaultValue string) (string, error) {
	value, err := r.Get(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (r propertyRepository) Delete(ctx context.Context, id string) error {
	return r.delete(ctx, Eq{"id": id})
}
