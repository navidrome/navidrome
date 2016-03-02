package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type Artist struct {
	BaseRepository
}

func NewArtistRepository() *Artist {
	r := &Artist{}
	r.init("artist", &domain.Artist{})
	return r
}

func (r *Artist) Put(m *domain.Artist) error {
	if m.Id == "" {
		m.Id = r.NewId(m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *Artist) Get(id string) (*domain.Artist, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Artist), err
}

func (r *Artist) GetByName(name string) (*domain.Artist, error) {
	id := r.NewId(name)
	return r.Get(id)
}

