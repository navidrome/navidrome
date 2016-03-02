package repositories

import (
	"github.com/deluan/gosonic/models"
"errors"
)

type Property interface {
	Put(id string, value string) error
	Get(id string) (string, error)
	DefaultGet(id string, defaultValue string) (string, error)
}

type property struct {
	BaseRepository
}

func NewPropertyRepository() *property {
	r := &property{}
	r.init("property", &models.Property{})
	return r
}

func (r *property) Put(id string, value string) error {
	m := &models.Property{Id: id, Value: value}
	if m.Id == "" {
		return errors.New("Id is required")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *property) Get(id string) (string, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.Property).Value, err
}

func (r*property) DefaultGet(id string, defaultValue string) (string, error) {
	v, err := r.Get(id)

	if v == "" {
		v = defaultValue
	}

	return v, err
}
