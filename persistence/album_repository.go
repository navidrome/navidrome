package persistence

import (
	"context"
	"time"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/consts"
	"github.com/deluan/navidrome/log"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/rest"
)

type albumRepository struct {
	sqlRepository
	sqlRestful
}

func NewAlbumRepository(ctx context.Context, o orm.Ormer) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "album"
	r.sortMappings = map[string]string{
		"artist": "compilation asc, album_artist asc, name asc",
		"random": "RANDOM()",
	}
	r.filterMappings = map[string]filterFunc{
		"name":        fullTextFilter,
		"compilation": booleanFilter,
		"artist_id":   artistFilter,
		"year":        yearFilter,
	}

	return r
}

func yearFilter(field string, value interface{}) Sqlizer {
	return Or{
		And{
			Gt{"min_year": 0},
			LtOrEq{"min_year": value},
			GtOrEq{"max_year": value},
		},
		Eq{"max_year": value},
	}
}

func artistFilter(field string, value interface{}) Sqlizer {
	return Exists("media_file", And{
		ConcatExpr("album_id=album.id"),
		Or{
			Eq{"artist_id": value},
			Eq{"album_artist_id": value},
		},
	})
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	return r.count(Select(), options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"id": id}))
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	return r.newSelectWithAnnotation("album.id", options...).Columns("*")
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"id": id})
	var res model.Album
	err := r.queryOne(sq, &res)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (r *albumRepository) FindByArtist(artistId string) (model.Albums, error) {
	sq := r.selectAlbum().Where(Eq{"album_artist_id": artistId}).OrderBy("max_year")
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	return res, err
}

// TODO Keep order when paginating
func (r *albumRepository) GetRandom(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	sq = sq.OrderBy("RANDOM()")
	results := model.Albums{}
	err := r.queryAll(sq, &results)
	return results, err
}

func (r *albumRepository) Refresh(ids ...string) error {
	type refreshAlbum struct {
		model.Album
		CurrentId   string
		HasCoverArt bool
		SongArtists string
	}
	var albums []refreshAlbum
	sel := Select(`album_id as id, album as name, f.artist, f.album_artist, f.artist_id, f.album_artist_id, 
		f.compilation, f.genre, max(f.year) as max_year, min(f.year) as min_year, sum(f.duration) as duration, 
		count(*) as song_count, a.id as current_id, f.id as cover_art_id, f.path as cover_art_path, 
		group_concat(f.artist, ' ') as song_artists, f.has_cover_art`).
		From("media_file f").
		LeftJoin("album a on f.album_id = a.id").
		Where(Eq{"f.album_id": ids}).GroupBy("album_id").OrderBy("f.id")
	err := r.queryAll(sel, &albums)
	if err != nil {
		return err
	}

	toInsert := 0
	toUpdate := 0
	for _, al := range albums {
		if !al.HasCoverArt {
			al.CoverArtId = ""
		}
		if al.Compilation {
			al.AlbumArtist = consts.VariousArtists
			al.AlbumArtistID = consts.VariousArtistsID
		}
		if al.AlbumArtist == "" {
			al.AlbumArtist = al.Artist
			al.AlbumArtistID = al.ArtistID
		}
		al.UpdatedAt = time.Now()
		if al.CurrentId != "" {
			toUpdate++
		} else {
			toInsert++
			al.CreatedAt = time.Now()
		}
		al.FullText = r.getFullText(al.Name, al.Artist, al.AlbumArtist, al.SongArtists)
		_, err := r.put(al.ID, al.Album)
		if err != nil {
			return err
		}
	}
	if toInsert > 0 {
		log.Debug(r.ctx, "Inserted new albums", "totalInserted", toInsert)
	}
	if toUpdate > 0 {
		log.Debug(r.ctx, "Updated albums", "totalUpdated", toUpdate)
	}
	return err
}

func (r *albumRepository) PurgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_id) from media_file)")
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged empty albums", "totalDeleted", c)
		}
	}
	return err
}

func (r *albumRepository) GetStarred(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...).Where("starred = true")
	starred := model.Albums{}
	err := r.queryAll(sq, &starred)
	return starred, err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	results := model.Albums{}
	err := r.doSearch(q, offset, size, &results, "name")
	return results, err
}

func (r *albumRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *albumRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *albumRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *albumRepository) EntityName() string {
	return "album"
}

func (r *albumRepository) NewInstance() interface{} {
	return &model.Album{}
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
