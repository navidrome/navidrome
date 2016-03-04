package persistence

import (
	"errors"
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
		return errors.New("Album Id is not set")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *albumRepository) Get(id string) (*domain.Album, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Album), err
}

func (r *albumRepository) FindByArtist(artistId string) (domain.Albums, error) {
	var as = make(domain.Albums, 0)
	err := r.loadChildren("artist", artistId, &as, domain.QueryOptions{SortBy: "Year"})
	return as, err
}

func (r *albumRepository) GetAll(options domain.QueryOptions) (domain.Albums, error) {
	var as = make(domain.Albums, 0)
	err := r.loadAll(&as, options)
	return as, err
}

var _ domain.AlbumRepository = (*albumRepository)(nil)
