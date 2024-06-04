package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	Tags         string `structs:"-" json:"-"`
}

func (a *dbAlbum) PostScan() error {
	if a.Discs != "" {
		if err := json.Unmarshal([]byte(a.Discs), &a.Album.Discs); err != nil {
			return err
		}
	}
	if a.Tags == "" {
		return nil
	}
	tags, err := parseTags(a.Tags)
	if err != nil {
		return err
	}
	if len(tags) != 0 {
		a.Album.Tags = tags
	}
	a.Album.Genre, a.Album.Genres = tags.ToGenres()
	return nil
}

func (a *dbAlbum) PostMapArgs(args map[string]any) error {
	delete(args, "tags")
	if len(a.Album.Discs) == 0 {
		args["discs"] = "{}"
		return nil
	}
	b, err := json.Marshal(a.Album.Discs)
	if err != nil {
		return err
	}
	args["discs"] = string(b)
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
		"genre_id":        tagIDFilter,
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
	sql = r.withTags(sql)
	return r.count(sql, options...)
}

func (r *albumRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"album.id": id}))
}

func (r *albumRepository) selectAlbum(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelectWithAnnotation("album.id", options...).Columns("album.*")
	sql = r.withTags(sql).GroupBy(r.tableName + ".id")
	//if len(options) > 0 && options[0].Filters != nil {
	//	s, _, _ := options[0].Filters.ToSql()
	//	// If there's any reference of genre in the filter, joins with genre
	//	if strings.Contains(s, "genre") {
	//		sql = r.withGenres(sql)
	// FIXME Genres
	//		// If there's no filter on genre_id, group the results by media_file.id
	//		if !strings.Contains(s, "genre_id") {
	//			sql = sql.GroupBy("album.id")
	//		}
	//	}
	//}
	return sql
}

func (r *albumRepository) Get(id string) (*model.Album, error) {
	sq := r.selectAlbum().Where(Eq{"album.id": id})
	var res dbAlbums
	if err := r.queryAll(sq, &res); err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return res[0].Album, nil
}

func (r *albumRepository) Put(al *model.Album) error {
	id, err := r.put(al.ID, &dbAlbum{Album: al})
	if err != nil {
		return err
	}
	al.ID = id
	return r.updateTags(al.ID, al.Tags)
}

func (r *albumRepository) GetAll(options ...model.QueryOptions) (model.Albums, error) {
	r.resetSeededRandom(options)
	sq := r.selectAlbum(options...)
	var dba dbAlbums
	err := r.queryAll(sq, &dba)
	if err != nil {
		return nil, err
	}
	return dba.toModels(), err
}

func (r *albumRepository) Touch(ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	upd := Update(r.tableName).Set("updated_at", time.Now()).Where(Eq{"id": ids})
	c, err := r.executeSQL(upd)
	if err == nil {
		log.Error(r.ctx, "Touching albums", "ids", ids, "updated", c == 1)
	}
	return err
}

func (r *albumRepository) GetOutdatedAlbumIDs(libID int) ([]string, error) {
	sel := r.newSelect().Columns("album.id").From("album").
		Join("library on library.id = album.library_id").
		Where(And{
			Eq{"library.id": libID},
			// FIXME This must be time the album was touched by the scanner
			ConcatExpr("album.updated_at > library.last_scan_started_at"),
		})
	var res []string
	err := r.queryAllSlice(sel, &res)
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
	var dba dbAlbums
	err := r.doSearch(q, offset, size, &dba, "name")
	if err != nil {
		return nil, err
	}
	return dba.toModels(), err
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
