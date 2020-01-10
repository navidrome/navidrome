package ledis

import (
	"errors"

	"github.com/cloudsonic/sonic-server/domain"
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
	if m.ID == "" {
		return errors.New("artist ID is not set")
	}
	return r.saveOrUpdate(m.ID, m)
}

func (r *artistRepository) Get(id string) (*domain.Artist, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Artist), err
}

func (r *artistRepository) PurgeInactive(active domain.Artists) ([]string, error) {
	return r.purgeInactive(active, func(e interface{}) string {
		return e.(domain.Artist).ID
	})
}

var _ domain.ArtistRepository = (*artistRepository)(nil)
