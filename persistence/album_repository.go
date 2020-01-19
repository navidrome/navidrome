package persistence

import (
	"fmt"
	"strings"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/cloudsonic/sonic-server/log"
	"github.com/cloudsonic/sonic-server/model"
)

type album struct {
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
	Genre        string    `orm:"index"`
	StarredAt    time.Time `orm:"index;null"`
	CreatedAt    time.Time `orm:"null"`
	UpdatedAt    time.Time `orm:"null"`
}

type albumRepository struct {
	searchableRepository
}

func NewAlbumRepository() model.AlbumRepository {
	r := &albumRepository{}
	r.tableName = "album"
	return r
}

func (r *albumRepository) Put(a *model.Album) error {
	ta := album(*a)
	return withTx(func(o orm.Ormer) error {
		return r.put(o, a.ID, a.Name, &ta)
	})
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	ta := album{ID: id}
	err := Db().Read(&ta)
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
	_, err := r.newQuery(Db()).Filter("artist_id", artistId).OrderBy("year", "name").All(&albums)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(albums), nil
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	var all []album
	_, err := r.newQuery(Db(), options...).All(&all)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(all), nil
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
	o := Db()
	sql := fmt.Sprintf(`
select album_id as id, album as name, f.artist, f.album_artist, f.artist_id, f.compilation, f.genre,  
	max(f.year) as year, sum(f.play_count) as play_count, max(f.play_date) as play_date,  sum(f.duration) as duration,
	max(f.updated_at) as updated_at, min(f.created_at) as created_at, count(*) as song_count, 
	a.id as current_id, f.id as cover_art_id, f.path as cover_art_path, f.has_cover_art
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
		err := r.addToIndex(o, r.tableName, al.ID, al.Name)
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
			// Don't update Starred/Rating
			_, err := o.Update(&al, "name", "artist_id", "cover_art_path", "cover_art_id", "artist", "album_artist",
				"year", "compilation", "play_count", "play_date", "song_count", "duration", "updated_at", "created_at")
			if err != nil {
				return err
			}
		}
		log.Debug("Updated albums", "num", len(toUpdate))
	}
	return err
}

func (r *albumRepository) PurgeInactive(activeList model.Albums) error {
	return withTx(func(o orm.Ormer) error {
		_, err := r.purgeInactive(o, activeList, func(item interface{}) string {
			return item.(model.Album).ID
		})
		return err
	})
}

func (r *albumRepository) PurgeEmpty() error {
	o := Db()
	_, err := o.Raw("delete from album where id not in (select distinct(album_id) from media_file)").Exec()
	return err
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	var starred []album
	_, err := r.newQuery(Db(), options...).Filter("starred", true).All(&starred)
	if err != nil {
		return nil, err
	}
	return r.toAlbums(starred), nil
}

func (r *albumRepository) SetStar(starred bool, ids ...string) error {
	if len(ids) == 0 {
		return model.ErrNotFound
	}
	var starredAt time.Time
	if starred {
		starredAt = time.Now()
	}
	_, err := r.newQuery(Db()).Filter("id__in", ids).Update(orm.Params{
		"starred":    starred,
		"starred_at": starredAt,
	})
	return err
}

func (r *albumRepository) MarkAsPlayed(id string, playDate time.Time) error {
	_, err := r.newQuery(Db()).Filter("id", id).Update(orm.Params{
		"play_count": orm.ColValue(orm.ColAdd, 1),
		"play_date":  playDate,
	})
	return err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	if len(q) <= 2 {
		return nil, nil
	}

	var results []album
	err := r.doSearch(r.tableName, q, offset, size, &results, "rating desc", "starred desc", "play_count desc", "name")
	if err != nil {
		return nil, err
	}
	return r.toAlbums(results), nil
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ = model.Album(album{})
