package db_sql

import (
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
)

type Playlist struct {
	ID       string `orm:"pk;column(id)"`
	Name     string `orm:"index"`
	Comment  string
	FullPath string
	Duration int
	Owner    string
	Public   bool
	Tracks   string
}

type playlistRepository struct {
	sqlRepository
}

func NewPlaylistRepository() domain.PlaylistRepository {
	r := &playlistRepository{}
	r.tableName = "playlist"
	return r
}

func (r *playlistRepository) Put(p *domain.Playlist) error {
	tp := r.fromDomain(p)
	return WithTx(func(o orm.Ormer) error {
		return r.put(o, p.ID, &tp)
	})
}

func (r *playlistRepository) Get(id string) (*domain.Playlist, error) {
	tp := &Playlist{ID: id}
	err := Db().Read(tp)
	if err == orm.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := r.toDomain(tp)
	return &a, err
}

func (r *playlistRepository) GetAll(options ...domain.QueryOptions) (domain.Playlists, error) {
	var all []Playlist
	_, err := r.newQuery(Db(), options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toPlaylists(all)
}

func (r *playlistRepository) toPlaylists(all []Playlist) (domain.Playlists, error) {
	result := make(domain.Playlists, len(all))
	for i, p := range all {
		result[i] = r.toDomain(&p)
	}
	return result, nil
}

func (r *playlistRepository) PurgeInactive(activeList domain.Playlists) ([]string, error) {
	return r.purgeInactive(activeList, func(item interface{}) string {
		return item.(domain.Playlist).ID
	})
}

func (r *playlistRepository) toDomain(p *Playlist) domain.Playlist {
	return domain.Playlist{
		ID:       p.ID,
		Name:     p.Name,
		Comment:  p.Comment,
		FullPath: p.FullPath,
		Duration: p.Duration,
		Owner:    p.Owner,
		Public:   p.Public,
		Tracks:   strings.Split(p.Tracks, ","),
	}
}

func (r *playlistRepository) fromDomain(p *domain.Playlist) Playlist {
	return Playlist{
		ID:       p.ID,
		Name:     p.Name,
		Comment:  p.Comment,
		FullPath: p.FullPath,
		Duration: p.Duration,
		Owner:    p.Owner,
		Public:   p.Public,
		Tracks:   strings.Join(p.Tracks, ","),
	}
}

var _ domain.PlaylistRepository = (*playlistRepository)(nil)
