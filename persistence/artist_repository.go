package persistence

import (
	"errors"

	"github.com/deluan/gosonic/domain"
)

type artistRepository struct {
	ledisRepository
}

func NewArtistRepository() domain.ArtistRepository {
	r := &artistRepository{}
	r.init("artist", &domain.Artist{})
	return r
}

func (r *artistRepository) Put(m *domain.Artist) error {
	if m.Id == "" {
		return errors.New("Artist Id is not set")
	}
	return r.saveOrUpdate(m.Id, m)
}

func (r *artistRepository) Get(id string) (*domain.Artist, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Artist), err
}

func (r *artistRepository) GetByName(name string) (*domain.Artist, error) {
	id := r.NewId(name)
	return r.Get(id)
}

func (r *artistRepository) PurgeInactive(active *domain.Artists) error {
	currentIds, err := r.getAllIds()
	if err != nil {
		return err
	}
	for _, a := range *active {
		currentIds[a.Id] = false
	}
	inactiveIds := make(map[string]bool)
	for id, inactive := range currentIds {
		if inactive {
			inactiveIds[id] = true
		}
	}
	return r.DeleteAll(inactiveIds)
}

var _ domain.ArtistRepository = (*artistRepository)(nil)
