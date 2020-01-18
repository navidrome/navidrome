package persistence

import (
	"fmt"
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
)

type Artist struct {
	ID         string `orm:"pk;column(id)"`
	Name       string `orm:"index"`
	AlbumCount int    `orm:"column(album_count)"`
}

type artistRepository struct {
	searchableRepository
}

func NewArtistRepository() model.ArtistRepository {
	r := &artistRepository{}
	r.tableName = "artist"
	return r
}

func (r *artistRepository) Put(a *model.Artist) error {
	ta := Artist(*a)
	return withTx(func(o orm.Ormer) error {
		return r.put(o, a.ID, a.Name, &ta)
	})
}

func (r *artistRepository) Get(id string) (*model.Artist, error) {
	ta := Artist{ID: id}
	err := Db().Read(&ta)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.Artist(ta)
	return &a, nil
}

func (r *artistRepository) Refresh(ids ...string) error {
	type refreshArtist struct {
		Artist
		CurrentId   string
		AlbumArtist string
		Compilation bool
	}
	var artists []refreshArtist
	o := Db()
	sql := fmt.Sprintf(`
select f.artist_id as id,
       f.artist as name,
       f.album_artist,
       f.compilation,
       count(*) as album_count,
       a.id as current_id
from album f
         left outer join artist a on f.artist_id = a.id
where f.artist_id in ('%s') group by f.artist_id order by f.id`, strings.Join(ids, "','"))
	_, err := o.Raw(sql).QueryRows(&artists)
	if err != nil {
		return err
	}

	var toInsert []Artist
	var toUpdate []Artist
	for _, al := range artists {
		if al.Compilation {
			al.AlbumArtist = "Various Artists"
		}
		if al.AlbumArtist != "" {
			al.Name = al.AlbumArtist
		}
		if al.CurrentId != "" {
			toUpdate = append(toUpdate, al.Artist)
		} else {
			toInsert = append(toInsert, al.Artist)
		}
	}
	if len(toInsert) > 0 {
		n, err := o.InsertMulti(100, toInsert)
		if err != nil {
			return err
		}
		log.Debug("Inserted new artists", "num", n)
	}
	if len(toUpdate) > 0 {
		for _, al := range toUpdate {
			_, err := o.Update(&al, "name", "album_count")
			if err != nil {
				return err
			}
		}
		log.Debug("Updated artists", "num", len(toUpdate))
	}
	return err
}

func (r *artistRepository) PurgeInactive(activeList model.Artists) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.purgeInactive(o, activeList, func(item interface{}) string {
			return item.(model.Artist).ID
		})
		return err
	})
}

func (r *artistRepository) Search(q string, offset int, size int) (model.Artists, error) {
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

func (r *artistRepository) toArtists(all []Artist) model.Artists {
	result := make(model.Artists, len(all))
	for i, a := range all {
		result[i] = model.Artist(a)
	}
	return result
}

var _ model.ArtistRepository = (*artistRepository)(nil)
var _ = model.Artist(Artist{})
