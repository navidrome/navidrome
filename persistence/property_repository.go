package persistence

import (
	"errors"

	"github.com/cloudsonic/sonic-server/domain"
	"github.com/cloudsonic/sonic-server/engine"
)

type propertyRepository struct {
	ledisRepository
}

func NewPropertyRepository() engine.PropertyRepository {
	r := &propertyRepository{}
	r.init("property", &engine.Property{})
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	m := &engine.Property{Id: id, Value: value}
	if m.Id == "" {
		return errors.New("Id is required")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *propertyRepository) Get(id string) (string, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*engine.Property).Value, err
}

func (r *propertyRepository) DefaultGet(id string, defaultValue string) (string, error) {
	v, err := r.Get(id)

	if err == domain.ErrNotFound {
		return defaultValue, nil
	}

	return v, err
}

var _ engine.PropertyRepository = (*propertyRepository)(nil)
