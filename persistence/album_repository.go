package persistence

import (
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
)

type album struct {
	ID           string    `json:"id"            orm:"pk;column(id)"`
	Name         string    `json:"name"          orm:"index"`
	ArtistID     string    `json:"artistId"      orm:"column(artist_id);index"`
	CoverArtPath string    `json:"-"`
	CoverArtId   string    `json:"-"`
	Artist       string    `json:"artist"        orm:"index"`
	AlbumArtist  string    `json:"albumArtist"`
	Year         int       `json:"year"          orm:"index"`
	Compilation  bool      `json:"compilation"`
	SongCount    int       `json:"songCount"`
	Duration     int       `json:"duration"`
	Genre        string    `json:"genre"         orm:"index"`
	CreatedAt    time.Time `json:"createdAt"     orm:"null"`
	UpdatedAt    time.Time `json:"updatedAt"     orm:"null"`
}

type albumRepository struct {
	searchableRepository
}

func NewAlbumRepository(o orm.Ormer) model.AlbumRepository {
	r := &albumRepository{}
	r.ormer = o
	r.tableName = "album"
	return r
}

func (r *albumRepository) Put(a *model.Album) error {
	ta := album(*a)
	return r.put(a.ID, a.Name, &ta)
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	ta := album{ID: id}
	err := r.ormer.Read(&ta)
	if err == orm.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a := model.Album(ta)
	return &a, err
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	var albums []album
	_, err := r.newQuery().Filter("artist_id", artistId).OrderBy("year", "name").All(&albums)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(albums), nil
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	var all []album
	_, err := r.newQuery(options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(all), nil
}

func (r *albumRepository) GetMap(ids []string) (map[string]model.Album, error) {
	var all []album
	if len(ids) == 0 {
		return nil, nil
	}
	_, err := r.newQuery().Filter("id__in", ids).All(&all)
	if err != nil {
		return nil, err
	}
	res := make(map[string]model.Album)
	for _, a := range all {
		res[a.ID] = model.Album(a)
	}
	return res, nil
}

// TODO Keep order when paginating
func (r *albumRepository) GetRandom(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.newRawQuery(options...)
	switch r.ormer.Driver().Type() {
	case orm.DRMySQL:
		sq = sq.OrderBy("RAND()")
	default:
		sq = sq.OrderBy("RANDOM()")
	}
	sql, args, err := sq.ToSql()
	if err != nil {
		return nil, err
	}
	var results []album
	_, err = r.ormer.Raw(sql, args...).QueryRows(&results)
	return r.toAlbums(results), err
}

func (r *albumRepository) toAlbums(all []album) model.Albums {
	result := make(model.Albums, len(all))
	for i, a := range all {
		result[i] = model.Album(a)
	}
	return result
}

func (r *albumRepository) Refresh(ids ...string) error {
	type refreshAlbum struct {
		album
		CurrentId   string
		HasCoverArt bool
	}
	var albums []refreshAlbum
	o := r.ormer
	sql := fmt.Sprintf(`
select album_id as id, album as name, f.artist, f.album_artist, f.artist_id, f.compilation, f.genre,  
	max(f.year) as year, sum(f.duration) as duration, max(f.updated_at) as updated_at, 
	min(f.created_at) as created_at, count(*) as song_count, a.id as current_id, f.id as cover_art_id, 
	f.path as cover_art_path, f.has_cover_art
from media_file f left outer join album a on f.album_id = a.id
where f.album_id in ('%s')
group by album_id order by f.id`, strings.Join(ids, "','"))
	_, err := o.Raw(sql).QueryRows(&albums)
	if err != nil {
		return err
	}

	var toInsert []album
	var toUpdate []album
	for _, al := range albums {
		if !al.HasCoverArt {
			al.CoverArtId = ""
		}
		if al.Compilation {
			al.AlbumArtist = "Various Artists"
		}
		if al.AlbumArtist == "" {
			al.AlbumArtist = al.Artist
		}
		if al.CurrentId != "" {
			toUpdate = append(toUpdate, al.album)
		} else {
			toInsert = append(toInsert, al.album)
		}
		err := r.addToIndex(r.tableName, al.ID, al.Name)
		if err != nil {
			return err
		}
	}
	if len(toInsert) > 0 {
		n, err := o.InsertMulti(10, toInsert)
		if err != nil {
			return err
		}
		log.Debug("Inserted new albums", "num", n)
	}
	if len(toUpdate) > 0 {
		for _, al := range toUpdate {
			_, err := o.Update(&al, "name", "artist_id", "cover_art_path", "cover_art_id", "artist", "album_artist",
				"year", "compilation", "song_count", "duration", "updated_at", "created_at")
			if err != nil {
				return err
			}
		}
		log.Debug("Updated albums", "num", len(toUpdate))
	}
	return err
}

func (r *albumRepository) PurgeEmpty() error {
	_, err := r.ormer.Raw("delete from album where id not in (select distinct(album_id) from media_file)").Exec()
	return err
}

func (r *albumRepository) GetStarred(userId string, options ...model.QueryOptions) (model.Albums, error) {
	var starred []album
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
	return r.toAlbums(starred), nil
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []album
	err := r.doSearch(r.tableName, q, offset, size, &results, "name")
	if err != nil {
		return nil, err
	}
	return r.toAlbums(results), nil
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ = model.Album(album{})
