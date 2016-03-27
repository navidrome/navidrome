package persistence

import (
	"errors"
	"sort"

	"github.com/deluan/gosonic/domain"
)

type artistIndexRepository struct {
	ledisRepository
}

func NewArtistIndexRepository() domain.ArtistIndexRepository {
	r := &artistIndexRepository{}
	r.init("index", &domain.ArtistIndex{})
	return r
}

func (r *artistIndexRepository) Put(m *domain.ArtistIndex) error {
	if m.Id == "" {
		return errors.New("index Id is not set")
	}
	sort.Sort(m.Artists)
	return r.saveOrUpdate(m.Id, m)
}

func (r *artistIndexRepository) Get(id string) (*domain.ArtistIndex, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.ArtistIndex), err
}

func (r *artistIndexRepository) GetAll() (domain.ArtistIndexes, error) {
	var indices = make(domain.ArtistIndexes, 0)
	err := r.loadAll(&indices, domain.QueryOptions{Alpha: true})
	return indices, err
}

func (r *artistIndexRepository) DeleteAll() error {
	var empty domain.ArtistIndexes
	_, err := r.purgeInactive(empty, func(e interface{}) string {
		return e.(domain.ArtistIndex).Id
	})

	return err
}

var _ domain.ArtistIndexRepository = (*artistIndexRepository)(nil)
