package db_storm

import (
	"github.com/cloudsonic/sonic-server/domain"
)

type _ArtistIndex struct {
	ID      string
	Artists domain.ArtistInfos
}

type artistIndexRepository struct {
	stormRepository
}

func NewArtistIndexRepository() domain.ArtistIndexRepository {
	r := &artistIndexRepository{}
	r.init(&_ArtistIndex{})
	return r
}

func (r *artistIndexRepository) Put(i *domain.ArtistIndex) error {
	ti := _ArtistIndex(*i)
	return Db().Save(&ti)
}

func (r *artistIndexRepository) Get(id string) (*domain.ArtistIndex, error) {
	ta := &_ArtistIndex{}
	err := r.getByID(id, ta)
	if err != nil {
		return nil, err
	}
	a := domain.ArtistIndex(*ta)
	return &a, err
}

func (r *artistIndexRepository) GetAll() (domain.ArtistIndexes, error) {
	var all []_ArtistIndex
	err := r.getAll(&all, &domain.QueryOptions{})
	if err != nil {
		return nil, err
	}
	return r.toArtistIndexes(all)
}

func (r *artistIndexRepository) toArtistIndexes(all []_ArtistIndex) (domain.ArtistIndexes, error) {
	result := make(domain.ArtistIndexes, len(all))
	for i, a := range all {
		result[i] = domain.ArtistIndex(a)
	}
	return result, nil
}

func (r *artistIndexRepository) DeleteAll() error {
	return Db().Drop(&_ArtistIndex{})
}

var _ domain.ArtistIndexRepository = (*artistIndexRepository)(nil)
var _ = domain.ArtistIndex(_ArtistIndex{})
