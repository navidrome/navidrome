package persistence

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/model"
)

type property struct {
	ID    string `orm:"pk;column(id)"`
	Value string
}

type propertyRepository struct {
	sqlRepository
}

func NewPropertyRepository(o orm.Ormer) model.PropertyRepository {
	r := &propertyRepository{}
	r.ormer = o
	r.tableName = "property"
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	p := &property{ID: id, Value: value}
	num, err := r.ormer.Update(p)
	if err != nil {
		return nil
	}
	if num == 0 {
		_, err = r.ormer.Insert(p)
	}
	return err
}

func (r *propertyRepository) Get(id string) (string, error) {
	p := &property{ID: id}
	err := r.ormer.Read(p)
	if err == orm.ErrNoRows {
		return "", model.ErrNotFound
	}
	return p.Value, err
}

func (r *propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	value, err := r.Get(id)
	if err == model.ErrNotFound {
		return defaultValue, nil
	}
	if err != nil {
		return defaultValue, err
	}
	return value, nil
}

var _ model.PropertyRepository = (*propertyRepository)(nil)
