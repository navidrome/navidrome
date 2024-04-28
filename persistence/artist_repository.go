package persistence

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
	"github.com/pocketbase/dbx"
)

type artistRepository struct {
	sqlRepository
	sqlRestful
	indexGroups utils.IndexGroups
}

type dbArtist struct {
	*model.Artist  `structs:",flatten"`
	SimilarArtists string `structs:"-" json:"similarArtists"`
}

func (a *dbArtist) PostScan() error {
	if a.SimilarArtists == "" {
		return nil
	}
	for _, s := range strings.Split(a.SimilarArtists, ";") {
		fields := strings.Split(s, ":")
		if len(fields) != 2 {
			continue
		}
		name, _ := url.QueryUnescape(fields[1])
		a.Artist.SimilarArtists = append(a.Artist.SimilarArtists, model.Artist{
			ID:   fields[0],
			Name: name,
		})
	}
	return nil
}
func (a *dbArtist) PostMapArgs(m map[string]any) error {
	var sa []string
	for _, s := range a.Artist.SimilarArtists {
		sa = append(sa, fmt.Sprintf("%s:%s", s.ID, url.QueryEscape(s.Name)))
	}
	m["similar_artists"] = strings.Join(sa, ";")
	return nil
}

func NewArtistRepository(ctx context.Context, db dbx.Builder) model.ArtistRepository {
	r := &artistRepository{}
	r.ctx = ctx
	r.db = db
	r.indexGroups = utils.ParseIndexGroups(conf.Server.IndexGroups)
	r.tableName = "artist"
	r.filterMappings = map[string]filterFunc{
		"id":      idFilter(r.tableName),
		"name":    fullTextFilter,
		"starred": booleanFilter,
	}
	if conf.Server.PreferSortTags {
		r.sortMappings = map[string]string{
			"name": "COALESCE(NULLIF(sort_artist_name,''),order_artist_name)",
		}
	} else {
		r.sortMappings = map[string]string{
			"name": "order_artist_name",
		}
	}
	return r
}

func (r *artistRepository) selectArtist(options ...model.QueryOptions) SelectBuilder {
	sql := r.newSelectWithAnnotation("artist.id", options...).Columns("artist.*")
	return r.withGenres(sql).GroupBy("artist.id")
}

func (r *artistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sql := r.newSelectWithAnnotation("artist.id")
	sql = r.withGenres(sql) // Required for filtering by genre
	return r.count(sql, options...)
}

func (r *artistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"artist.id": id}))
}

func (r *artistRepository) Put(a *model.Artist, colsToUpdate ...string) error {
	a.FullText = getFullText(a.Name, a.SortArtistName)
	dba := &dbArtist{Artist: a}
	_, err := r.put(dba.ID, dba, colsToUpdate...)
	if err != nil {
		return err
	}
	if a.ID == consts.VariousArtistsID {
		return r.updateGenres(a.ID, nil)
	}
	return r.updateGenres(a.ID, a.Genres)
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
	err := loadAllGenres(r, res)
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
	err = loadAllGenres(r, res)
	return res, err
}

func (r *artistRepository) toModels(dba []dbArtist) model.Artists {
	res := model.Artists{}
	for i := range dba {
		res = append(res, *dba[i].Artist)
	}
	return res
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
