package storm

import (
	"reflect"

	"github.com/asdine/storm"
	"github.com/asdine/storm/index"
	"github.com/asdine/storm/q"
	"github.com/cloudsonic/sonic-server/domain"
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

func (r *stormRepository) extractID(record interface{}) string {
	v := reflect.ValueOf(record).Elem()
	id := v.FieldByName("ID").String()
	return id
}

func (r *stormRepository) getByID(id string, ta interface{}) error {
	err := Db().One("ID", id, ta)
	if err == storm.ErrNotFound {
		return domain.ErrNotFound
	}
	return nil
}

func (r *stormRepository) purgeInactive(ids []string) (deleted []string, err error) {
	query := Db().Select(q.Not(q.In("ID", ids)))

	// Collect IDs that will be deleted
	err = query.Each(r.bucket, func(record interface{}) error {
		id := r.extractID(record)
		deleted = append(deleted, id)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(deleted) == 0 {
		return
	}

	err = query.Delete(r.bucket)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

func (r *stormRepository) execute(matcher q.Matcher, result *[]_Album, options ...*domain.QueryOptions) error {
	query := Db().Select(matcher)
	if len(options) > 0 {
		query = addQueryOptions(query, options[0])
	}
	err := query.Find(result)
	if err == storm.ErrNotFound {
		return nil
	}
	return err
}

func stormOptions(options *domain.QueryOptions) func(*index.Options) {
	return func(opts *index.Options) {
		opts.Reverse = options.Desc
		opts.Skip = options.Offset
		if options.Size > 0 {
			opts.Limit = options.Size
		}
	}
}

func addQueryOptions(q storm.Query, o *domain.QueryOptions) storm.Query {
	if o.SortBy != "" {
		q = q.OrderBy(o.SortBy)
	}
	if o.Desc {
		q = q.Reverse()
	}
	if o.Size > 0 {
		q = q.Limit(o.Size)
	}
	return q.Skip(o.Offset)
}
