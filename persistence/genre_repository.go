package persistence

import (
	"context"

	. "github.com/Masterminds/squirrel"
	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
	"github.com/pocketbase/dbx"
)

type genreRepository struct {
	*baseTagRepository
}

func NewGenreRepository(ctx context.Context, db dbx.Builder) model.GenreRepository {
	genreFilter := model.TagGenre
	return &genreRepository{
		baseTagRepository: newBaseTagRepository(ctx, db, &genreFilter),
	}
}

func (r *genreRepository) selectGenre(opt ...model.QueryOptions) SelectBuilder {
	return r.newSelect(opt...).Columns("tag.tag_value as name")
}

func (r *genreRepository) GetAll(opt ...model.QueryOptions) (model.Genres, error) {
	sq := r.selectGenre(opt...)
	res := model.Genres{}
	err := r.queryAll(sq, &res)
	return res, err
}

// Override ResourceRepository methods to return Genre objects instead of Tag objects

func (r *genreRepository) Read(id string) (interface{}, error) {
	sel := r.selectGenre().Where(Eq{"tag.id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

func (r *genreRepository) ReadAll(options ...rest.QueryOptions) (interface{}, error) {
	return r.GetAll(r.parseRestOptions(r.ctx, options...))
}

func (r *genreRepository) NewInstance() interface{} {
	return &model.Genre{}
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ model.ResourceRepository = (*genreRepository)(nil)
