package persistence

import (
	"context"
	"fmt"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
)

type albumRepository struct {
	sqlRepository
	sqlRestful
}

func NewAlbumRepository(ctx context.Context, o orm.QueryExecutor) model.AlbumRepository {
	r := &albumRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "album"
	r.sortMappings = map[string]string{
		"name":           "order_album_name asc, order_album_artist_name asc",
		"artist":         "compilation asc, order_album_artist_name asc, order_album_name asc",
		"random":         "RANDOM()",
		"max_year":       "coalesce(nullif(original_date,''), cast(max_year as text)), release_date, name, order_album_name asc",
		"recently_added": recentlyAddedSort(),
	}
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

	return r
}

func recentlyAddedSort() string {
	if conf.Server.RecentlyAddedByModTime {
		return "updated_at"
	}
	return "created_at"
}

func recentlyPlayedFilter(field string, value interface{}) Sqlizer {
	return Gt{"play_count": 0}
}

func hasRatingFilter(field string, value interface{}) Sqlizer {
	return Gt{"rating": 0}
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
	var res model.Albums
	if err := r.queryAll(sq, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	err := r.loadAlbumGenres(&res)
	return &res[0], err
}

func (r *albumRepository) Put(m *model.Album) error {
	_, err := r.put(m.ID, m)
	if err != nil {
		return err
	}
	return r.updateGenres(m.ID, r.tableName, m.Genres)
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	res, err := r.GetAllWithoutGenres(options...)
	if err != nil {
		return nil, err
	}
	err = r.loadAlbumGenres(&res)
	return res, err
}

func (r *albumRepository) GetAllWithoutGenres(options ...model.QueryOptions) (model.Albums, error) {
	sq := r.selectAlbum(options...)
	res := model.Albums{}
	err := r.queryAll(sq, &res)
	if err != nil {
		return nil, err
	}
	return res, err
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
	results := model.Albums{}
	err := r.doSearch(q, offset, size, &results, "name")
	if err != nil {
		return nil, err
	}
	err = r.loadAlbumGenres(&results)
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
