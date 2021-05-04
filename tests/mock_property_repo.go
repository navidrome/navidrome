package tests

import "github.com/navidrome/navidrome/model"

type mockedPropertyRepo struct {
	model.UserRepository
	data map[string]string
	err  error
}

func (p *mockedPropertyRepo) init() {
	if p.data == nil {
		p.data = make(map[string]string)
	}
}

func (p *mockedPropertyRepo) Put(id string, value string) error {
	if p.err != nil {
		return p.err
	}
	p.init()
	p.data[id] = value
	return nil
}

func (p *mockedPropertyRepo) Get(id string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	p.init()
	if v, ok := p.data[id]; ok {
		return v, nil
	}
	return "", model.ErrNotFound
}

func (p *mockedPropertyRepo) DefaultGet(id string, defaultValue string) (string, error) {
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
