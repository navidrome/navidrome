package repositories

import (
	"github.com/deluan/gosonic/models"
"errors"
)

type ArtistIndex struct {
	BaseRepository
}

func NewArtistIndexRepository() *ArtistIndex {
	r := &ArtistIndex{}
	r.table = "index"
	return r
}

func (r *ArtistIndex) Put(m *models.ArtistIndex) error {
	if m.Id == "" {
		return errors.New("Id is not set")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r*ArtistIndex) Get(id string) (*models.ArtistIndex, error) {
	entity := &models.ArtistIndex{}
	err := r.loadEntity(id, entity)
	return entity, err
}

func (r*ArtistIndex) GetAll() ([]*models.ArtistIndex, error) {
	return nil, errors.New("Not Implemented")
}


