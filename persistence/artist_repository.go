package persistence

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/conf"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
	"github.com/cloudsonic/sonic-server/utils"
)

type artist struct {
	ID         string `json:"id"         orm:"pk;column(id)"`
	Name       string `json:"name"       orm:"index"`
	AlbumCount int    `json:"albumCount" orm:"column(album_count)"`
}

type artistRepository struct {
	searchableRepository
	indexGroups utils.IndexGroups
}

func NewArtistRepository(o orm.Ormer) model.ArtistRepository {
	r := &artistRepository{}
	r.ormer = o
	r.indexGroups = utils.ParseIndexGroups(conf.Sonic.IndexGroups)
	r.tableName = "artist"
	return r
}

func (r *artistRepository) getIndexKey(a *artist) string {
	name := strings.ToLower(utils.NoArticle(a.Name))
	for k, v := range r.indexGroups {
		key := strings.ToLower(k)
		if strings.HasPrefix(name, key) {
			return v
		}
	}
	return "#"
}

func (r *artistRepository) Put(a *model.Artist) error {
	ta := artist(*a)
	return r.put(a.ID, a.Name, &ta)
}

func (r *artistRepository) Get(id string) (*model.Artist, error) {
	ta := artist{ID: id}
	err := r.ormer.Read(&ta)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.Artist(ta)
	return &a, nil
}

// TODO Cache the index (recalculate when there are changes to the DB)
func (r *artistRepository) GetIndex() (model.ArtistIndexes, error) {
	var all []artist
	// TODO Paginate
	_, err := r.newQuery().OrderBy("name").All(&all)
	if err != nil {
		return nil, err
	}

	fullIdx := make(map[string]*model.ArtistIndex)
	for _, a := range all {
		ax := r.getIndexKey(&a)
		idx, ok := fullIdx[ax]
		if !ok {
			idx = &model.ArtistIndex{ID: ax}
			fullIdx[ax] = idx
		}
		idx.Artists = append(idx.Artists, model.Artist(a))
	}
	var result model.ArtistIndexes
	for _, idx := range fullIdx {
		result = append(result, *idx)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func (r *artistRepository) Refresh(ids ...string) error {
	type refreshArtist struct {
		artist
		CurrentId   string
		AlbumArtist string
		Compilation bool
	}
	var artists []refreshArtist
	o := r.ormer
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

	var toInsert []artist
	var toUpdate []artist
	for _, ar := range artists {
		if ar.Compilation {
			ar.AlbumArtist = "Various Artists"
		}
		if ar.AlbumArtist != "" {
			ar.Name = ar.AlbumArtist
		}
		if ar.CurrentId != "" {
			toUpdate = append(toUpdate, ar.artist)
		} else {
			toInsert = append(toInsert, ar.artist)
		}
		err := r.addToIndex(r.tableName, ar.ID, ar.Name)
		if err != nil {
			return err
		}
	}
	if len(toInsert) > 0 {
		n, err := o.InsertMulti(10, toInsert)
		if err != nil {
			return err
		}
		log.Debug("Inserted new artists", "num", n)
	}
	if len(toUpdate) > 0 {
		for _, al := range toUpdate {
			// Don't update Starred
			_, err := o.Update(&al, "name", "album_count")
			if err != nil {
				return err
			}
		}
		log.Debug("Updated artists", "num", len(toUpdate))
	}
	return err
}

func (r *artistRepository) GetStarred(userId string, options ...model.QueryOptions) (model.Artists, error) {
	var starred []artist
	sq := r.newRawQuery(options...).Join("annotation").Where("annotation.item_id = " + r.tableName + ".id")
	sq = sq.Where(squirrel.And{
		squirrel.Eq{"annotation.user_id": userId},
		squirrel.Eq{"annotation.starred": true},
	})
	sql, args, err := sq.ToSql()
	if err != nil {
		return nil, err
	}
	_, err = r.ormer.Raw(sql, args...).QueryRows(&starred)
	if err != nil {
		return nil, err
	}
	return r.toArtists(starred), nil
}

func (r *artistRepository) SetStar(starred bool, ids ...string) error {
	if len(ids) == 0 {
		return model.ErrNotFound
	}
	var starredAt time.Time
	if starred {
		starredAt = time.Now()
	}
	_, err := r.newQuery().Filter("id__in", ids).Update(orm.Params{
		"starred":    starred,
		"starred_at": starredAt,
	})
	return err
}

func (r *artistRepository) PurgeEmpty() error {
	_, err := r.ormer.Raw("delete from artist where id not in (select distinct(artist_id) from album)").Exec()
	return err
}

func (r *artistRepository) Search(q string, offset int, size int) (model.Artists, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []artist
	err := r.doSearch(r.tableName, q, offset, size, &results, "name")
	if err != nil {
		return nil, err
	}

	return r.toArtists(results), nil
}

func (r *artistRepository) toArtists(all []artist) model.Artists {
	result := make(model.Artists, len(all))
	for i, a := range all {
		result[i] = model.Artist(a)
	}
	return result
}

var _ model.ArtistRepository = (*artistRepository)(nil)
var _ = model.Artist(artist{})
