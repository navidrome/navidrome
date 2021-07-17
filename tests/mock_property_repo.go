package tests

import "github.com/navidrome/navidrome/model"

type MockedPropertyRepo struct {
	model.PropertyRepository
	data map[string]string
	err  error
}

func (p *MockedPropertyRepo) init() {
	if p.data == nil {
		p.data = make(map[string]string)
	}
}

func (p *MockedPropertyRepo) Put(id string, value string) error {
	if p.err != nil {
		return p.err
	}
	p.init()
	p.data[id] = value
	return nil
}

func (p *MockedPropertyRepo) Get(id string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	p.init()
	if v, ok := p.data[id]; ok {
		return v, nil
	}
	return "", model.ErrNotFound
}

func (p *MockedPropertyRepo) Delete(id string) error {
	if p.err != nil {
		return p.err
	}
	p.init()
	if _, ok := p.data[id]; ok {
		delete(p.data, id)
		return nil
	}
	return model.ErrNotFound
}

func (p *MockedPropertyRepo) DefaultGet(id string, defaultValue string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	p.init()
	v, err := p.Get(id)
	if err != nil {
		return defaultValue, nil
	}
	return v, nil
}
