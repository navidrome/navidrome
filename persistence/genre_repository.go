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

func NewGenreRepository(db dbx.Builder) model.GenreRepository {
	return &genreRepository{
		baseTagRepository: newBaseTagRepository(db, new(model.TagGenre)),
	}
}

func (r *genreRepository) selectGenre(ctx context.Context, opt ...model.QueryOptions) SelectBuilder {
	return r.newSelect(ctx, opt...).Columns("tag.tag_value as name")
}

func (r *genreRepository) GetAll(ctx context.Context, opt ...model.QueryOptions) (model.Genres, error) {
	sq := r.selectGenre(ctx, opt...)
	res := model.Genres{}
	err := r.queryAll(ctx, sq, &res)
	return res, err
}

func (r *genreRepository) Get(ctx context.Context, id string) (*model.Genre, error) {
	sel := r.selectGenre(ctx).Where(Eq{"tag.id": id})
	var res model.Genre
	err := r.queryOne(ctx, sel, &res)
	return &res, err
}

// Override the base tag REST methods to return Genre objects instead of Tag objects

func (r *genreRepository) Read(ctx context.Context, id string) (*model.Genre, error) {
	return r.Get(ctx, id)
}

func (r *genreRepository) ReadAll(ctx context.Context, options ...rest.QueryOptions) ([]model.Genre, error) {
	return r.GetAll(ctx, r.parseRestOptions(ctx, options...))
}

var _ model.GenreRepository = (*genreRepository)(nil)
var _ rest.Repository[model.Genre] = (*genreRepository)(nil)
