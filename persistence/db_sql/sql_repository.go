package db_sql

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/persistence"
)

type sqlRepository struct {
	entityName string
}

func (r *sqlRepository) newQuery(o orm.Ormer) orm.QuerySeter {
	return o.QueryTable(r.entityName)
}

func (r *sqlRepository) CountAll() (int64, error) {
	return r.newQuery(Db()).Count()
}

func (r *sqlRepository) Exists(id string) (bool, error) {
	c, err := r.newQuery(Db()).Filter("id", id).Count()
	return c == 1, err
}

func (r *artistRepository) put(id string, a interface{}) error {
	return WithTx(func(o orm.Ormer) error {
		c, err := r.newQuery(Db()).Filter("id", id).Count()
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
		qs := o.QueryTable(r.entityName).Exclude("id__in", ids)
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
