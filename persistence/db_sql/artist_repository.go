package db_sql

import (
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/domain"
)

// This is used to isolate Storm's struct tags from the domain, to keep it agnostic of persistence details
type Artist struct {
	ID         string `orm:"pk;column(id)"`
	Name       string `orm:"index"`
	AlbumCount int    `orm:"column(album_count)"`
}

type artistRepository struct {
	sqlRepository
}

func NewArtistRepository() domain.ArtistRepository {
	r := &artistRepository{}
	r.tableName = "artist"
	return r
}

func (r *artistRepository) Put(a *domain.Artist) error {
	ta := Artist(*a)
	return WithTx(func(o orm.Ormer) error {
		return r.put(o, a.ID, &ta)
	})
}

func (r *artistRepository) Get(id string) (*domain.Artist, error) {
	ta := Artist{ID: id}
	err := Db().Read(&ta)
	if err == orm.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := domain.Artist(ta)
	return &a, nil
}

func (r *artistRepository) PurgeInactive(activeList domain.Artists) ([]string, error) {
	return r.purgeInactive(activeList, func(item interface{}) string {
		return item.(domain.Artist).ID
	})
}

var _ domain.ArtistRepository = (*artistRepository)(nil)
var _ = domain.Artist(Artist{})
