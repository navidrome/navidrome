package tests

import (
	"context"

	"github.com/deluan/rest"
	"github.com/navidrome/navidrome/model"
)

type MockedGenreRepo struct {
	Error   error
	Data    map[string]model.Genre
	Options model.QueryOptions
}

func (r *MockedGenreRepo) init() {
	if r.Data == nil {
		r.Data = make(map[string]model.Genre)
	}
}

func (r *MockedGenreRepo) GetAll(_ context.Context, options ...model.QueryOptions) (model.Genres, error) {
	if len(options) > 0 {
		r.Options = options[0]
	}
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()

	var all model.Genres
	for _, g := range r.Data {
		all = append(all, g)
	}
	return all, nil
}

func (r *MockedGenreRepo) Get(_ context.Context, id string) (*model.Genre, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()
	if g, ok := r.Data[id]; ok {
		return &g, nil
	}
	return nil, model.ErrNotFound
}

func (r *MockedGenreRepo) Put(g *model.Genre) error {
	if r.Error != nil {
		return r.Error
	}
	r.init()
	r.Data[g.ID] = *g
	return nil
}

func (r *MockedGenreRepo) Count(context.Context, ...rest.QueryOptions) (int64, error) {
	if r.Error != nil {
		return 0, r.Error
	}
	r.init()
	return int64(len(r.Data)), nil
}

func (r *MockedGenreRepo) Read(ctx context.Context, id string) (*model.Genre, error) {
	return r.Get(ctx, id)
}

func (r *MockedGenreRepo) ReadAll(ctx context.Context, _ ...rest.QueryOptions) ([]model.Genre, error) {
	return r.GetAll(ctx)
}

var _ model.GenreRepository = (*MockedGenreRepo)(nil)
