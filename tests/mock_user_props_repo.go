package tests

import "github.com/navidrome/navidrome/model"

type MockedUserPropsRepo struct {
	model.UserPropsRepository
	Error error
	Data  map[string]string
}

func (p *MockedUserPropsRepo) init() {
	if p.Data == nil {
		p.Data = make(map[string]string)
	}
}

func (p *MockedUserPropsRepo) Put(userId, key string, value string) error {
	if p.Error != nil {
		return p.Error
	}
	p.init()
	p.Data[userId+key] = value
	return nil
}

func (p *MockedUserPropsRepo) Get(userId, key string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	if v, ok := p.Data[userId+key]; ok {
		return v, nil
	}
	return "", model.ErrNotFound
}

func (p *MockedUserPropsRepo) Delete(userId, key string) error {
	if p.Error != nil {
		return p.Error
	}
	p.init()
	if _, ok := p.Data[userId+key]; ok {
		delete(p.Data, userId+key)
		return nil
	}
	return model.ErrNotFound
}

func (p *MockedUserPropsRepo) DefaultGet(userId, key string, defaultValue string) (string, error) {
	if p.Error != nil {
		return "", p.Error
	}
	p.init()
	v, err := p.Get(userId, key)
	if err != nil {
		return defaultValue, nil //nolint:nilerr
	}
	return v, nil
}
