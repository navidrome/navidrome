package db_ledis

import (
	"errors"

	"github.com/cloudsonic/sonic-server/domain"
)

type playlistRepository struct {
	ledisRepository
}

func NewPlaylistRepository() domain.PlaylistRepository {
	r := &playlistRepository{}
	r.init("playlist", &domain.Playlist{})
	return r
}

func (r *playlistRepository) Put(m *domain.Playlist) error {
	if m.ID == "" {
		return errors.New("playlist ID is not set")
	}
	return r.saveOrUpdate(m.ID, m)
}

func (r *playlistRepository) Get(id string) (*domain.Playlist, error) {
	var rec interface{}
	rec, err := r.readEntity(id)
	return rec.(*domain.Playlist), err
}

func (r *playlistRepository) GetAll(options ...domain.QueryOptions) (domain.Playlists, error) {
	o := domain.QueryOptions{}
	if len(options) > 0 {
		o = options[0]
	}
	var as = make(domain.Playlists, 0)
	if o.SortBy == "" {
		o.SortBy = "Name"
		o.Alpha = true
	}
	err := r.loadAll(&as, o)
	return as, err
}

func (r *playlistRepository) PurgeInactive(active domain.Playlists) ([]string, error) {
	return r.purgeInactive(active, func(e interface{}) string {
		return e.(domain.Playlist).ID
	})
}

var _ domain.PlaylistRepository = (*playlistRepository)(nil)
