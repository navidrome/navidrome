package db_sql

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
)

type Property struct {
	ID    string `orm:"pk;column(id)"`
	Value string
}

type propertyRepository struct {
	sqlRepository
}

func NewPropertyRepository() domain.PropertyRepository {
	r := &propertyRepository{}
	r.tableName = "property"
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	p := &Property{ID: id, Value: value}
	num, err := Db().Update(p)
	if err != nil {
		return nil
	}
	if num == 0 {
		_, err = Db().Insert(p)
	}
	return err
}

func (r *propertyRepository) Get(id string) (string, error) {
	p := &Property{ID: id}
	err := Db().Read(p)
	if err == orm.ErrNoRows {
		return "", domain.ErrNotFound
	}
	return p.Value, err
}

func (r *propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	value, err := r.Get(id)
	if err == domain.ErrNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

var _ domain.PropertyRepository = (*propertyRepository)(nil)
