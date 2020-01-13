package db_sql

import (
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
)

type Album struct {
	ID           string    `orm:"pk;column(id)"`
	Name         string    `orm:"index"`
	ArtistID     string    `orm:"column(artist_id);index"`
	CoverArtPath string    ``
	CoverArtId   string    ``
	Artist       string    `orm:"index"`
	AlbumArtist  string    ``
	Year         int       `orm:"index"`
	Compilation  bool      ``
	Starred      bool      `orm:"index"`
	PlayCount    int       `orm:"index"`
	PlayDate     time.Time `orm:"null;index"`
	SongCount    int       ``
	Duration     int       ``
	Rating       int       `orm:"index"`
	Genre        string    ``
	StarredAt    time.Time `orm:"null"`
	CreatedAt    time.Time `orm:"null"`
	UpdatedAt    time.Time `orm:"null"`
}

type albumRepository struct {
	sqlRepository
}

func NewAlbumRepository() domain.AlbumRepository {
	r := &albumRepository{}
	r.tableName = "album"
	return r
}

func (r *albumRepository) Put(a *domain.Album) error {
	ta := Album(*a)
	return r.put(a.ID, &ta)
}

func (r *albumRepository) Get(id string) (*domain.Album, error) {
	ta := Album{ID: id}
	err := Db().Read(&ta)
	if err == orm.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := domain.Album(ta)
	return &a, err
}

func (r *albumRepository) FindByArtist(artistId string) (domain.Albums, error) {
	var albums []Album
	_, err := r.newQuery(Db()).Filter("artist_id", artistId).OrderBy("year", "name").All(&albums)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(albums)
}

func (r *albumRepository) GetAll(options ...domain.QueryOptions) (domain.Albums, error) {
	var all []Album
	_, err := r.newQuery(Db(), options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(all)
}

func (r *albumRepository) toAlbums(all []Album) (domain.Albums, error) {
	result := make(domain.Albums, len(all))
	for i, a := range all {
		result[i] = domain.Album(a)
	}
	return result, nil
}

func (r *albumRepository) PurgeInactive(activeList domain.Albums) ([]string, error) {
	return r.purgeInactive(activeList, func(item interface{}) string {
		return item.(domain.Album).ID
	})
}

func (r *albumRepository) GetStarred(options ...domain.QueryOptions) (domain.Albums, error) {
	var starred []Album
	_, err := r.newQuery(Db(), options...).Filter("starred", true).All(&starred)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(starred)
}

var _ domain.AlbumRepository = (*albumRepository)(nil)
var _ = domain.Album(Album{})
