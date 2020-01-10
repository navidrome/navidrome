package storm

import (
	"reflect"

	"github.com/asdine/storm"
	"github.com/asdine/storm/q"
)

type stormRepository struct {
	bucket interface{}
}

func (r *stormRepository) init(entity interface{}) {
	r.bucket = entity
	if err := Db().Init(r.bucket); err != nil {
		panic(err)
	}
	if err := Db().ReIndex(r.bucket); err != nil {
		panic(err)
	}
}

func (r *stormRepository) CountAll() (int64, error) {
	c, err := Db().Count(r.bucket)
	return int64(c), err
}

func (r *stormRepository) Exists(id string) (bool, error) {
	err := Db().One("ID", id, r.bucket)
	if err != nil {
		return false, err
	}
	return err != storm.ErrNotFound, nil
}

func (r *stormRepository) purgeInactive(ids []string) (deleted []string, err error) {
	query := Db().Select(q.Not(q.In("ID", ids)))

	err = query.Each(r.bucket, func(record interface{}) error {
		v := reflect.ValueOf(record).Elem()
		id := v.FieldByName("ID").String()
		deleted = append(deleted, id)
		return nil
	})
	if err != nil {
		return nil, err
	}

	err = query.Delete(r.bucket)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}
