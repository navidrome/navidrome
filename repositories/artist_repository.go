package repositories

import (
	"github.com/deluan/gosonic/models"
)

type Artist struct {
	BaseRepository
}

func NewArtistRepository() *Artist {
	r := &Artist{}
	r.table = "artist"
	return r
}

func (r *Artist) Put(m *models.Artist) error {
	if m.Id == "" {
		m.Id = r.NewId(m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *Artist) Get(id string) (*models.Artist, error) {
	rec := &models.Artist{}
	err := r.loadEntity(id, rec)
	return rec, err
}

func (r *Artist) GetByName(name string) (*models.Artist, error) {
	id := r.NewId(name)
	return r.Get(id)
}

