package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type genreRepository struct {
	sqlRepository
}

func NewGenreRepository(ctx context.Context, db dbx.Builder) model.GenreRepository {
	r := &genreRepository{}
	r.ctx = ctx
	r.db = db
	r.registerModel(&model.Tag{}, map[string]filterFunc{
		"name": containsFilter("tag_value"),
	})
	r.setSortMappings(map[string]string{
		"name": "tag_name",
	})
	return r
}

func (r *genreRepository) selectGenre(opt ...model.QueryOptions) SelectBuilder {
	return r.newSelect(opt...).
		Columns(
			"id",
			"tag_value as name",
			"album_count",
			"media_file_count as song_count",
		).
		Where(Eq{"tag.tag_name": model.TagGenre})
}

func (r *genreRepository) GetAll(opt ...model.QueryOptions) (model.Genres, error) {
	sq := r.selectGenre(opt...)
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

func (r *genreRepository) Count(options ...rest.QueryOptions) (int64, error) {
	return r.count(r.selectGenre(), r.parseRestOptions(r.ctx, options...))
}

func (r *genreRepository) Read(id string) (interface{}, error) {
	sel := r.selectGenre().Columns("*").Where(Eq{"id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *genreRepository) EntityName() string {
	return r.tableName
}

func (r *genreRepository) NewInstance() interface{} {
	return &model.Genre{}
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ model.ResourceRepository = (*genreRepository)(nil)
