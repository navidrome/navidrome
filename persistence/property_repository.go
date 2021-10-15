package persistence

import (
	"context"
	"errors"
	"strconv"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type propertyRepository struct {
	sqlRepository
}

func NewPropertyRepository(ctx context.Context, db dbx.Builder) model.PropertyRepository {
	r := &propertyRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "property"
	return r
}

func (r propertyRepository) Put(id string, value string) error {
	update := Update(r.tableName).Set("value", value).Where(Eq{"id": id})
	count, err := r.executeSQL(update)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	insert := Insert(r.tableName).Columns("id", "value").Values(id, value)
	_, err = r.executeSQL(insert)
	return err
}

func (r propertyRepository) Get(id string) (string, error) {
	sel := Select("value").From(r.tableName).Where(Eq{"id": id})
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
	if errors.Is(err, model.ErrNotFound) {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

func (r propertyRepository) DefaultGetBool(id string, defaultValue bool) (bool, error) {
	val, err := r.DefaultGet(id, strconv.FormatBool(defaultValue))
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(val)
}

func (r propertyRepository) Delete(id string) error {
	return r.delete(Eq{"id": id})
}

func (r propertyRepository) DeletePrefixed(prefix string) error {
	return r.delete(Like{"id": strings.Replace(prefix, "%", "%%", -1) + "%"})
}
