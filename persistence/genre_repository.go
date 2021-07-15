package persistence

import (
	"context"

	"github.com/deluan/rest"

	. "github.com/Masterminds/squirrel"
	"github.com/astaxie/beego/orm"
	"github.com/navidrome/navidrome/model"
)

type genreRepository struct {
	sqlRepository
	sqlRestful
}

func NewGenreRepository(ctx context.Context, o orm.Ormer) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.ormer = o
	r.tableName = "genre"
	return r
}

func (r *genreRepository) GetAll() (model.Genres, error) {
	sq := Select("genre as name", "count(distinct album_id) as album_count", "count(distinct id) as song_count").
		From("media_file").GroupBy("genre")
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *genreRepository) Put(m *model.Genre) error {
	_, err := r.put(m.ID, m)
	return err
}

func (r *genreRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(Select(), r.parseRestOptions(options...))
}

func (r *genreRepository) Read(id string) (interface{}, error) {
	sel := r.newSelect().Columns("*").Where(Eq{"id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	sel := r.newSelect(r.parseRestOptions(options...)).Columns("*")
	res := model.Genres{}
	err := r.queryAll(sel, &res)
	return res, err
}

func (r *genreRepository) EntityName() string {
	return r.tableName
}

func (r *genreRepository) NewInstance() interface{} {
	return &model.Genre{}
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ model.ResourceRepository = (*genreRepository)(nil)
