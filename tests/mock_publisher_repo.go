package tests

import (
	"github.com/navidrome/navidrome/model"
)

type MockedPublisherRepo struct {
	Error error
	data  map[string]model.Publisher
}

func (r *MockedPublisherRepo) init() {
	if r.data == nil {
		r.data = make(map[string]model.Publisher)
	}
}

func (r *MockedPublisherRepo) GetAll(...model.QueryOptions) (model.Publishers, error) {
	if r.Error != nil {
		return nil, r.Error
	}
	r.init()

	var all model.Publishers
	for _, g := range r.data {
		all = append(all, g)
	}
	return all, nil
}

func (r *MockedPublisherRepo) Put(g *model.Publisher) error {
	if r.Error != nil {
		return r.Error
	}
	r.init()
	r.data[g.ID] = *g
	return nil
}
