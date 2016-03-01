package repositories

import (
	"github.com/deluan/gosonic/models"
)

type Album struct {
	BaseRepository
}

func NewAlbumRepository() *Album {
	r := &Album{}
	r.init("album", &models.Album{})
	return r
}

func (r *Album) Put(m *models.Album) error {
	if m.Id == "" {
		m.Id = r.NewId(m.ArtistId, m.Name)
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *Album) Get(id string) (*models.Album, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*models.Album), err
}