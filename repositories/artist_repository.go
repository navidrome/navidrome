package repositories

import (
	"github.com/deluan/gosonic/models"
)

type Artist struct {
	BaseRepository
}

func NewArtistRepository() *Artist {
	r := &Artist{}
	r.init("artist", &models.Artist{})
	return r
}

func (r *Artist) Put(m *models.Artist) error {
	if m.Id == "" {
		m.Id = r.NewId(m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *Artist) Get(id string) (*models.Artist, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.Artist), err
}

func (r *Artist) GetByName(name string) (*models.Artist, error) {
	id := r.NewId(name)
	return r.Get(id)
}

