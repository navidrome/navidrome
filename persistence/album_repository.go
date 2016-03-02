package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type albumRepository struct {
	baseRepository
}

func NewAlbumRepository() domain.AlbumRepository {
	r := &albumRepository{}
	r.init("album", &domain.Album{})
	return r
}

func (r *albumRepository) Put(m *domain.Album) error {
	if m.Id == "" {
		m.Id = r.NewId(m.ArtistId, m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *albumRepository) Get(id string) (*domain.Album, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Album), err
}