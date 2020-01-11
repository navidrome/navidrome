package storm

import (
	"github.com/cloudsonic/sonic-server/domain"
)

// This is used to isolate Storm's struct tags from the domain, to keep it agnostic of persistence details
type _Artist struct {
	ID         string
	Name       string `storm:"index"`
	AlbumCount int
}

type artistRepository struct {
	stormRepository
}

func NewArtistRepository() domain.ArtistRepository {
	r := &artistRepository{}
	r.init(&_Artist{})
	return r
}

func (r *artistRepository) Put(a *domain.Artist) error {
	ta := _Artist(*a)
	return Db().Save(&ta)
}

func (r *artistRepository) Get(id string) (*domain.Artist, error) {
	ta := &_Artist{}
	err := r.getByID(id, ta)
	if err != nil {
		return nil, err
	}
	a := domain.Artist(*ta)
	return &a, nil
}

func (r *artistRepository) PurgeInactive(activeList domain.Artists) ([]string, error) {
	return r.purgeInactive(activeList)
}

var _ domain.ArtistRepository = (*artistRepository)(nil)
var _ = domain.Artist(_Artist{})
