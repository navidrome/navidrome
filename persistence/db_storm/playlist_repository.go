package db_storm

import (
	"github.com/cloudsonic/sonic-server/domain"
)

type _Playlist struct {
	ID       string
	Name     string
	Comment  string
	FullPath string
	Duration int
	Owner    string
	Public   bool
	Tracks   []string
}

type playlistRepository struct {
	stormRepository
}

func NewPlaylistRepository() domain.PlaylistRepository {
	r := &playlistRepository{}
	r.init(&_Playlist{})
	return r
}

func (r *playlistRepository) Put(p *domain.Playlist) error {
	tp := _Playlist(*p)
	return Db().Save(&tp)
}

func (r *playlistRepository) Get(id string) (*domain.Playlist, error) {
	tp := &_Playlist{}
	err := r.getByID(id, tp)
	if err != nil {
		return nil, err
	}
	a := domain.Playlist(*tp)
	return &a, err
}

func (r *playlistRepository) GetAll(options ...domain.QueryOptions) (domain.Playlists, error) {
	var all []_Playlist
	err := r.getAll(&all, options...)
	if err != nil {
		return nil, err
	}
	return r.toPlaylists(all)
}

func (r *playlistRepository) toPlaylists(all []_Playlist) (domain.Playlists, error) {
	result := make(domain.Playlists, len(all))
	for i, p := range all {
		result[i] = domain.Playlist(p)
	}
	return result, nil
}

func (r *playlistRepository) PurgeInactive(activeList domain.Playlists) ([]string, error) {
	return r.purgeInactive(activeList)
}

var _ domain.PlaylistRepository = (*playlistRepository)(nil)
var _ = domain.Playlist(_Playlist{})
