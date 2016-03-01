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
	r.init("index", &models.ArtistIndex{})
	return r
}

func (r *ArtistIndex) Put(m *models.ArtistIndex) error {
	if m.Id == "" {
		return errors.New("Id is not set")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *ArtistIndex) Get(id string) (*models.ArtistIndex, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.ArtistIndex), err
}

func (r *ArtistIndex) GetAll() ([]models.ArtistIndex, error) {
	var indices = make([]models.ArtistIndex, 30)
	err := r.loadAll(&indices)
	return indices, err
}


