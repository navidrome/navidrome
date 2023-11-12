package persistence

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/beego/beego/v2/client/orm"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

type artistRepository struct {
	sqlRepository
	sqlRestful
	indexGroups utils.IndexGroups
}

type dbArtist struct {
	model.Artist   `structs:",flatten"`
	SimilarArtists string `structs:"similar_artists" json:"similarArtists"`
}

func NewArtistRepository(ctx context.Context, o orm.QueryExecutor) model.ArtistRepository {
	r := &artistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.indexGroups = utils.ParseIndexGroups(conf.Server.IndexGroups)
	r.tableName = "artist"
	r.sortMappings = map[string]string{
		"name": "order_artist_name",
	}
	r.filterMappings = map[string]filterFunc{
		"id":      idFilter(r.tableName),
		"name":    fullTextFilter,
		"starred": booleanFilter,
	}
	return r
}

func (r *artistRepository) selectArtist(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelectWithAnnotation("artist.id", options...).Columns("artist.*")
	return r.withGenres(sql).GroupBy("artist.id")
}

func (r *artistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelectWithAnnotation("artist.id")
	sql = r.withGenres(sql)
	return r.count(sql, options...)
}

func (r *artistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"artist.id": id}))
}

func (r *artistRepository) Put(a *model.Artist, colsToUpdate ...string) error {
	a.FullText = getFullText(a.Name, a.SortArtistName)
	dba := r.fromModel(a)
	_, err := r.put(dba.ID, dba, colsToUpdate...)
	if err != nil {
		return err
	}
	if a.ID == consts.VariousArtistsID {
		return r.updateGenres(a.ID, r.tableName, nil)
	}
	return r.updateGenres(a.ID, r.tableName, a.Genres)
}

func (r *artistRepository) Get(id string) (*model.Artist, error) {
	sel := r.selectArtist().Where(Eq{"artist.id": id})
	var dba []dbArtist
	if err := r.queryAll(sel, &dba); err != nil {
		return nil, err
	}
	if len(dba) == 0 {
		return nil, model.ErrNotFound
	}
	res := r.toModels(dba)
	err := r.loadArtistGenres(&res)
	return &res[0], err
}

func (r *artistRepository) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	sel := r.selectArtist(options...)
	var dba []dbArtist
	err := r.queryAll(sel, &dba)
	if err != nil {
		return nil, err
	}
	res := r.toModels(dba)
	err = r.loadArtistGenres(&res)
	return res, err
}

func (r *artistRepository) toModels(dba []dbArtist) model.Artists {
	res := model.Artists{}
	for i := range dba {
		a := dba[i]
		res = append(res, *r.toModel(&a))
	}
	return res
}

func (r *artistRepository) toModel(dba *dbArtist) *model.Artist {
	a := dba.Artist
	a.SimilarArtists = nil
	for _, s := range strings.Split(dba.SimilarArtists, ";") {
		fields := strings.Split(s, ":")
		if len(fields) != 2 {
			continue
		}
		name, _ := url.QueryUnescape(fields[1])
		a.SimilarArtists = append(a.SimilarArtists, model.Artist{
			ID:   fields[0],
			Name: name,
		})
	}
	return &a
}

func (r *artistRepository) fromModel(a *model.Artist) *dbArtist {
	dba := &dbArtist{Artist: *a}
	var sa []string

	for _, s := range a.SimilarArtists {
		sa = append(sa, fmt.Sprintf("%s:%s", s.ID, url.QueryEscape(s.Name)))
	}

	dba.SimilarArtists = strings.Join(sa, ";")
	return dba
}

func (r *artistRepository) getIndexKey(a *model.Artist) string {
	name := strings.ToLower(utils.NoArticle(a.Name))
	for k, v := range r.indexGroups {
		key := strings.ToLower(k)
		if strings.HasPrefix(name, key) {
			return v
		}
	}
	return "#"
}

// TODO Cache the index (recalculate when there are changes to the DB)
func (r *artistRepository) GetIndex() (model.ArtistIndexes, error) {
	all, err := r.GetAll(model.QueryOptions{Sort: "order_artist_name"})
	if err != nil {
		return nil, err
	}

	fullIdx := make(map[string]*model.ArtistIndex)
	for i := range all {
		a := all[i]
		ax := r.getIndexKey(&a)
		idx, ok := fullIdx[ax]
		if !ok {
			idx = &model.ArtistIndex{ID: ax}
			fullIdx[ax] = idx
		}
		idx.Artists = append(idx.Artists, a)
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

func (r *artistRepository) purgeEmpty() error {
	del := Delete(r.tableName).Where("id not in (select distinct(album_artist_id) from album)")
	c, err := r.executeSQL(del)
	if err == nil {
		if c > 0 {
			log.Debug(r.ctx, "Purged empty artists", "totalDeleted", c)
		}
	}
	return err
}

func (r *artistRepository) Search(q string, offset int, size int) (model.Artists, error) {
	var dba []dbArtist
	err := r.doSearch(q, offset, size, &dba, "name")
	if err != nil {
		return nil, err
	}
	return r.toModels(dba), nil
}

func (r *artistRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.CountAll(r.parseRestOptions(options...))
}

func (r *artistRepository) Read(id string) (interface{}, error) {
	return r.Get(id)
}

func (r *artistRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(options...))
}

func (r *artistRepository) EntityName() string {
	return "artist"
}

func (r *artistRepository) NewInstance() interface{} {
	return &model.Artist{}
}

var _ model.ArtistRepository = (*artistRepository)(nil)
var _ model.ResourceRepository = (*artistRepository)(nil)
