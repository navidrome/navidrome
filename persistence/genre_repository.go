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
	return &genreRepository{
		baseTagRepository: newBaseTagRepository(ctx, db, new(model.TagGenre)),
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

func (r *genreRepository) Get(id string) (*model.Genre, error) {
	sel := r.selectGenre().Where(Eq{"tag.id": id})
	var res model.Genre
	err := r.queryOne(sel, &res)
	return &res, err
}

// Override the base tag REST methods to return Genre objects instead of Tag objects

func (r *genreRepository) Read(ctx context.Context, id string) (*model.Genre, error) {
	return r.Get(id)
}

func (r *genreRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Genre, error) {
	return r.GetAll(r.parseRestOptions(ctx, options...))
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ rest.Repository[model.Genre] = (*genreRepository)(nil)
