package persistence

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
	searchableRepository
}

func NewArtistRepository() domain.ArtistRepository {
	r := &artistRepository{}
	r.tableName = "artist"
	return r
}

func (r *artistRepository) Put(a *domain.Artist) error {
	ta := Artist(*a)
	return withTx(func(o orm.Ormer) error {
		return r.put(o, a.ID, a.Name, &ta)
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

func (r *artistRepository) PurgeInactive(activeList domain.Artists) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.purgeInactive(o, activeList, func(item interface{}) string {
			return item.(domain.Artist).ID
		})
		return err
	})
}

func (r *artistRepository) Search(q string, offset int, size int) (domain.Artists, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []Artist
	err := r.doSearch(r.tableName, q, offset, size, &results, "name")
	if err != nil {
		return nil, err
	}

	return r.toArtists(results), nil
}

func (r *artistRepository) toArtists(all []Artist) domain.Artists {
	result := make(domain.Artists, len(all))
	for i, a := range all {
		result[i] = domain.Artist(a)
	}
	return result
}

var _ domain.ArtistRepository = (*artistRepository)(nil)
var _ = domain.Artist(Artist{})
