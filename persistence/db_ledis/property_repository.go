package db_ledis

import (
	"errors"

	"github.com/cloudsonic/sonic-server/domain"
)

type propertyRepository struct {
	ledisRepository
}

func NewPropertyRepository() domain.PropertyRepository {
	r := &propertyRepository{}
	r.init("property", &domain.Property{})
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	m := &domain.Property{ID: id, Value: value}
	if m.ID == "" {
		return errors.New("ID is required")
	}
	return r.saveOrUpdate(m.ID, m)
}

func (r *propertyRepository) Get(id string) (string, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Property).Value, err
}

func (r *propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	v, err := r.Get(id)

	if err == domain.ErrNotFound {
		return defaultValue, nil
	}

	return v, err
}

var _ domain.PropertyRepository = (*propertyRepository)(nil)
