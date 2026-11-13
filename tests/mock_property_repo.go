package tests

import (
	"context"

	"github.com/navidrome/navidrome/model"
)

type MockedPropertyRepo struct {
	model.PropertyRepository
	Error error
	Data  map[string]string
}

func (p *MockedPropertyRepo) init() {
	if p.Data == nil {
		p.Data = make(map[string]string)
	}
}

func (p *MockedPropertyRepo) Put(_ context.Context, id string, value string) error {
	if p.Error != nil {
		return p.Error
	}
	p.init()
	p.Data[id] = value
	return nil
}

func (p *MockedPropertyRepo) Get(_ context.Context, id string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	if v, ok := p.Data[id]; ok {
		return v, nil
	}
	return "", model.ErrNotFound
}

func (p *MockedPropertyRepo) Delete(_ context.Context, id string) error {
	if p.Error != nil {
		return p.Error
	}
	p.init()
	if _, ok := p.Data[id]; ok {
		delete(p.Data, id)
		return nil
	}
	return model.ErrNotFound
}

func (p *MockedPropertyRepo) DefaultGet(ctx context.Context, id string, defaultValue string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	v, err := p.Get(ctx, id)
	if err != nil {
		return defaultValue, nil //nolint:nilerr
	}
	return v, nil
}
