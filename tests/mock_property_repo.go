package tests

import "github.com/navidrome/navidrome/model"

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

func (p *MockedPropertyRepo) Put(id string, value string) error {
	if p.Error != nil {
		return p.Error
	}
	p.init()
	p.Data[id] = value
	return nil
}

func (p *MockedPropertyRepo) Get(id string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	if v, ok := p.Data[id]; ok {
		return v, nil
	}
	return "", model.ErrNotFound
}

func (p *MockedPropertyRepo) Delete(id string) error {
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

func (p *MockedPropertyRepo) DefaultGet(id string, defaultValue string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	v, err := p.Get(id)
	if err != nil {
		return defaultValue, nil //nolint:nilerr
	}
	return v, nil
}
