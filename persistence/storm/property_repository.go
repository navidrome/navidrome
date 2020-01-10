package storm

import (
	"github.com/asdine/storm"
	"github.com/cloudsonic/sonic-server/domain"
)

const propertyBucket = "Property"

type propertyRepository struct {
}

func NewPropertyRepository() domain.PropertyRepository {
	r := &propertyRepository{}
	return r
}

func (r *propertyRepository) Put(id string, value string) error {
	return Db().Set(propertyBucket, id, value)
}

func (r *propertyRepository) Get(id string) (string, error) {
	var value string
	err := Db().Get(propertyBucket, id, &value)
	if err == storm.ErrNotFound {
		return value, domain.ErrNotFound
	}
	return value, err
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
