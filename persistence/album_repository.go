package persistence

import (
	"github.com/deluan/gosonic/domain"
)

type albumRepository struct {
	ledisRepository
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

func (r *albumRepository) FindByArtist(artistId string) ([]domain.Album, error) {
	var as = make([]domain.Album, 0)
	err := r.loadChildren("artist", artistId, &as, "Year", false)
	return as, err
}

var _ domain.AlbumRepository = (*albumRepository)(nil)