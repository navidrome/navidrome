package repositories

import (
	"github.com/deluan/gosonic/models"
"errors"
)

type Property interface {
	Put(id string, value string) error
	Get(id string) (string, error)
}

type PropertyImpl struct {
	BaseRepository
}

func NewPropertyRepository() *PropertyImpl {
	r := &PropertyImpl{}
	r.init("property", &models.Property{})
	return r
}

func (r *PropertyImpl) Put(id string, value string) error {
	m := &models.Property{Id: id, Value: value}
	if m.Id == "" {
		return errors.New("Id is required")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *PropertyImpl) Get(id string) (string, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.Property).Value, err
}
