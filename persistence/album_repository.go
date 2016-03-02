package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type Album struct {
	BaseRepository
}

func NewAlbumRepository() *Album {
	r := &Album{}
	r.init("album", &domain.Album{})
	return r
}

func (r *Album) Put(m *domain.Album) error {
	if m.Id == "" {
		m.Id = r.NewId(m.ArtistId, m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *Album) Get(id string) (*domain.Album, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Album), err
}