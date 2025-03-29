package tests

import (
	"github.com/navidrome/navidrome/model"
)

type MockedGenreRepo struct {
	Error error
	Data  map[string]model.Genre
}

func (r *MockedGenreRepo) init() {
	if r.Data == nil {
		r.Data = make(map[string]model.Genre)
	}
}

func (r *MockedGenreRepo) GetAll(...model.QueryOptions) (model.Genres, error) {
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

func (r *MockedGenreRepo) Put(g *model.Genre) error {
	if r.Error != nil {
		return r.Error
	}
	r.init()
	r.Data[g.ID] = *g
	return nil
}
