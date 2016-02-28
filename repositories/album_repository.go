package repositories

import (
	"github.com/deluan/gosonic/models"
)

type Album struct {
	BaseRepository
}

func NewAlbumRepository() *Album {
	r := &Album{}
	r.key = "album"
	return r
}

func (r *Album) Put(m *models.Album) (*models.Album, error) {
	if m.Id == "" {
		m.Id = r.NewId(m.Name)
	}
	return m, r.saveOrUpdate(m.Id, m)
}

func (r *Album) Get(id string) (*models.Album, error) {
	rec := &models.Album{}
	err := readStruct(r.key, id, rec)
	return rec, err
}

func (r *Album) GetByName(name string) (*models.Album, error) {
	id := r.NewId(name)
	return r.Get(id)
}

