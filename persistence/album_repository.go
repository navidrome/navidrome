package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type albumRepository struct {
	sqlRepository
	sqlRestful
}

type dbAlbum struct {
	*model.Album `structs:",flatten"`
	Discs        string `structs:"-" json:"discs"`
}

func (a *dbAlbum) PostScan() error {
	if a.Discs != "" {
		return json.Unmarshal([]byte(a.Discs), &a.Album.Discs)
	}
	return nil
}

func (a *dbAlbum) PostMapArgs(m map[string]any) error {
	if len(a.Album.Discs) == 0 {
		m["discs"] = "{}"
		return nil
	}
	b, err := json.Marshal(a.Album.Discs)
	if err != nil {
		return err
	}
	m["discs"] = string(b)
	return nil
}

type dbAlbums []dbAlbum

func (dba dbAlbums) toModels() model.Albums {
	res := make(model.Albums, len(dba))
	for i := range dba {
		res[i] = *dba[i].Album
	}
	return res
}

func NewAlbumRepository(ctx context.Context, db dbx.Builder) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.db = db
	r.tableName = "album"
	r.filterMappings = map[string]filterFunc{
		"id":              idFilter(r.tableName),
		"name":            fullTextFilter,
		"compilation":     booleanFilter,
		"artist_id":       artistFilter,
		"year":            yearFilter,
		"recently_played": recentlyPlayedFilter,
		"starred":         booleanFilter,
		"has_rating":      hasRatingFilter,
	}
	if conf.Server.PreferSortTags {
		r.sortMappings = map[string]string{
			"name":           "COALESCE(NULLIF(sort_album_name,''),order_album_name)",
			"artist":         "compilation asc, COALESCE(NULLIF(sort_album_artist_name,''),order_album_artist_name) asc, COALESCE(NULLIF(sort_album_name,''),order_album_name) asc",
			"albumArtist":    "compilation asc, COALESCE(NULLIF(sort_album_artist_name,''),order_album_artist_name) asc, COALESCE(NULLIF(sort_album_name,''),order_album_name) asc",
			"max_year":       "coalesce(nullif(original_date,''), cast(max_year as text)), release_date, name, COALESCE(NULLIF(sort_album_name,''),order_album_name) asc",
			"random":         r.seededRandomSort(),
			"recently_added": recentlyAddedSort(),
		}
	} else {
		r.sortMappings = map[string]string{
			"name":           "order_album_name asc, order_album_artist_name asc",
			"artist":         "compilation asc, order_album_artist_name asc, order_album_name asc",
			"albumArtist":    "compilation asc, order_album_artist_name asc, order_album_name asc",
			"max_year":       "coalesce(nullif(original_date,''), cast(max_year as text)), release_date, name, order_album_name asc",
			"random":         r.seededRandomSort(),
			"recently_added": recentlyAddedSort(),
		}
	}

	return r
}

func recentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "updated_at"
	}
	return "created_at"
}

func recentlyPlayedFilter(string, interface{}) Sqlizer {
	return Gt{"play_count": 0}
}

func hasRatingFilter(string, interface{}) Sqlizer {
	return Gt{"rating": 0}
}

func yearFilter(_ string, value interface{}) Sqlizer {
	return Or{
		And{
			Gt{"min_year": 0},
			LtOrEq{"min_year": value},
			GtOrEq{"max_year": value},
		},
		Eq{"max_year": value},
	}
}

func artistFilter(_ string, value interface{}) Sqlizer {
	return Like{"all_artist_ids": fmt.Sprintf("%%%s%%", value)}
}

func (r *albumRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelectWithAnnotation("album.id")
	sql = r.withGenres(sql) // Required for filtering by genre
	return r.count(sql, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"album.id": id}))
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelectWithAnnotation("album.id", options...).Columns("album.*")
	if len(options) > 0 && options[0].Filters != nil {
		s, _, _ := options[0].Filters.ToSql()
		// If there's any reference of genre in the filter, joins with genre
		if strings.Contains(s, "genre") {
			sql = r.withGenres(sql)
			// If there's no filter on genre_id, group the results by media_file.id
			if !strings.Contains(s, "genre_id") {
				sql = sql.GroupBy("album.id")
			}
		}
	}
	return sql
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"album.id": id})
	var dba dbAlbums
	if err := r.queryAll(sq, &dba); err != nil {
		return nil, err
	}
	if len(dba) == 0 {
		return nil, model.ErrNotFound
	}
	res := dba.toModels()
	err := loadAllGenres(r, res)
	return &res[0], err
}

func (r *albumRepository) Put(m *model.Album) error {
	_, err := r.put(m.ID, &dbAlbum{Album: m})
	if err != nil {
		return err
	}
	return r.updateGenres(m.ID, m.Genres)
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	res, err := r.GetAllWithoutGenres(options...)
	if err != nil {
		return nil, err
	}
	err = loadAllGenres(r, res)
	return res, err
}

func (r *albumRepository) GetAllWithoutGenres(options ...model.QueryOptions) (model.Albums, error) {
	r.resetSeededRandom(options)
	sq := r.selectAlbum(options...)
	var dba dbAlbums
	err := r.queryAll(sq, &dba)
	if err != nil {
		return nil, err
	}
	return dba.toModels(), err
}

func (r *albumRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_id) from media_file)")
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged empty albums", "totalDeleted", c)
		}
	}
	return err
}

func (r *albumRepository) Search(q string, offset int, size int) (model.Albums, error) {
	var dba dbAlbums
	err := r.doSearch(q, offset, size, &dba, "name")
	if err != nil {
		return nil, err
	}
	res := dba.toModels()
	err = loadAllGenres(r, res)
	return res, err
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

func (r *albumRepository) DeleteMany(ids ...string) error {
	return r.delete(Eq{"album.id": ids})
}

var _ model.AlbumRepository = (*albumRepository)(nil)
var _ model.ResourceRepository = (*albumRepository)(nil)
