package persistence

import (
	"errors"
	"github.com/deluan/gosonic/domain"
)

type propertyRepository struct {
	baseRepository
}

func NewPropertyRepository() domain.PropertyRepository {
	r := &propertyRepository{}
	r.init("property", &domain.Property{})
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	m := &domain.Property{Id: id, Value: value}
	if m.Id == "" {
		return errors.New("Id is required")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *propertyRepository) Get(id string) (string, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Property).Value, err
}

func (r *propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	v, err := r.Get(id)

	if v == "" {
		v = defaultValue
	}

	return v, err
}

var _ domain.PropertyRepository = (*propertyRepository)(nil)