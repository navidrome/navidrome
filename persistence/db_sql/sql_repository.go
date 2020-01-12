package db_sql

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/persistence"
)

type sqlRepository struct {
	entityName string
}

func (r *sqlRepository) newQuery(o orm.Ormer, options ...domain.QueryOptions) orm.QuerySeter {
	q := o.QueryTable(r.entityName)
	if len(options) > 0 {
		opts := options[0]
		q = q.Offset(opts.Offset)
		if opts.Size > 0 {
			q = q.Limit(opts.Size)
		}
		if opts.SortBy != "" {
			if opts.Desc {
				q = q.OrderBy("-" + opts.SortBy)
			} else {
				q = q.OrderBy(opts.SortBy)
			}
		}
	}
	return q
}

func (r *sqlRepository) CountAll() (int64, error) {
	return r.newQuery(Db()).Count()
}

func (r *sqlRepository) Exists(id string) (bool, error) {
	c, err := r.newQuery(Db()).Filter("id", id).Count()
	return c == 1, err
}

func (r *sqlRepository) put(id string, a interface{}) error {
	return WithTx(func(o orm.Ormer) error {
		c, err := r.newQuery(o).Filter("id", id).Count()
		if err != nil {
			return err
		}
		if c == 0 {
			_, err = o.Insert(a)
			return err
		}
		_, err = o.Update(a)
		return err
	})
}

func (r *sqlRepository) purgeInactive(activeList interface{}, getId func(item interface{}) string) ([]string, error) {
	ids := persistence.CollectValue(activeList, getId)
	var values []orm.Params
	err := WithTx(func(o orm.Ormer) error {
		qs := r.newQuery(o).Exclude("id__in", ids)
		num, err := qs.Values(&values, "id")
		if num > 0 {
			_, err = qs.Delete()
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v["ID"].(string)
	}
	return result, nil
}
