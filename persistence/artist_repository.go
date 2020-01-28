package persistence

import (
	"context"
	"sort"
	"strings"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/deluan/navidrome/conf"
	"github.com/deluan/navidrome/model"
	"github.com/deluan/navidrome/utils"
	"github.com/deluan/rest"
)

type artistRepository struct {
	sqlRepository
	indexGroups utils.IndexGroups
}

func NewArtistRepository(ctx context.Context, o orm.Ormer) model.ArtistRepository {
	r := &artistRepository{}
	r.ctx = ctx
	r.ormer = o
	r.indexGroups = utils.ParseIndexGroups(conf.Server.IndexGroups)
	r.tableName = "media_file"
	return r
}

func (r *artistRepository) selectArtist(options ...model.QueryOptions) SelectBuilder {
	// FIXME Handle AlbumArtist/Various Artists...
	return r.newSelectWithAnnotation(model.ArtistItemType, "album_id", options...).
		Columns("artist_id as id", "artist as name", "count(distinct album_id) as album_count").
		GroupBy("artist_id").Where(Eq{"compilation": false})
}

func (r *artistRepository) CountAll(options ...model.QueryOptions) (int64, error) {
	sel := r.selectArtist(options...).Where(Eq{"compilation": false})
	return r.count(sel, options...)
}

func (r *artistRepository) Exists(id string) (bool, error) {
	return r.exists(Select().Where(Eq{"artist_id": id}))
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

func (r *artistRepository) Put(a *model.Artist) error {
	return nil
}

func (r *artistRepository) Get(id string) (*model.Artist, error) {
	sel := Select("artist_id as id", "artist as name", "count(distinct album_id) as album_count").
		From("media_file").GroupBy("artist_id").Where(Eq{"artist_id": id})
	var res model.Artist
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *artistRepository) GetAll(options ...model.QueryOptions) (model.Artists, error) {
	sel := r.selectArtist(options...)
	var res model.Artists
	err := r.queryAll(sel, &res)
	return res, err
}

// TODO Cache the index (recalculate when there are changes to the DB)
func (r *artistRepository) GetIndex() (model.ArtistIndexes, error) {
	sq := Select("artist_id as id", "artist as name", "count(distinct album_id) as album_count").
		From("media_file").GroupBy("artist_id").OrderBy("name")
	var all model.Artists
	// TODO Paginate
	err := r.queryAll(sq, &all)
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
	return nil
}

func (r *artistRepository) GetStarred(userId string, options ...model.QueryOptions) (model.Artists, error) {
	return nil, nil // TODO
}

func (r *artistRepository) PurgeEmpty() error {
	return nil
}

func (r *artistRepository) Search(q string, offset int, size int) (model.Artists, error) {
	return nil, nil // TODO
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
var _ model.ArtistRepository = (*artistRepository)(nil)
var _ model.ResourceRepository = (*artistRepository)(nil)
