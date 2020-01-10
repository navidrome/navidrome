package ledis

import (
	"errors"
	"time"

	"github.com/cloudsonic/sonic-server/domain"
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
	if m.ID == "" {
		return errors.New("album ID is not set")
	}
	return r.saveOrUpdate(m.ID, m)
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

func (r *albumRepository) GetAllIds() ([]string, error) {
	idMap, err := r.getAllIds()
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(idMap))

	i := 0
	for id := range idMap {
		ids[i] = id
		i++
	}

	return ids, nil
}

func (r *albumRepository) PurgeInactive(active domain.Albums) ([]string, error) {
	return r.purgeInactive(active, func(e interface{}) string {
		return e.(domain.Album).ID
	})
}

func (r *albumRepository) GetStarred(options domain.QueryOptions) (domain.Albums, error) {
	var as = make(domain.Albums, 0)
	start := time.Time{}.Add(1 * time.Hour)
	err := r.loadRange("Starred", start, time.Now(), &as, options)
	return as, err
}

var _ domain.AlbumRepository = (*albumRepository)(nil)
