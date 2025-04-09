package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/model/id"
)

type MockedRadioRepo struct {
	model.RadioRepository
	Data    map[string]*model.Radio
	All     model.Radios
	Err     bool
	Options model.QueryOptions
}

func CreateMockedRadioRepo() *MockedRadioRepo {
	return &MockedRadioRepo{}
}

func (m *MockedRadioRepo) SetError(err bool) {
	m.Err = err
}

func (m *MockedRadioRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.Err {
		return 0, errors.New("error")
	}
	return int64(len(m.Data)), nil
}

func (m *MockedRadioRepo) Delete(id string) error {
	if m.Err {
		return errors.New("Error!")
	}

	_, found := m.Data[id]

	if !found {
		return errors.New("not found")
	}

	delete(m.Data, id)
	return nil
}

func (m *MockedRadioRepo) Exists(id string) (bool, error) {
	if m.Err {
		return false, errors.New("Error!")
	}
	_, found := m.Data[id]
	return found, nil
}

func (m *MockedRadioRepo) Get(id string) (*model.Radio, error) {
	if m.Err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.Data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockedRadioRepo) GetAll(qo ...model.QueryOptions) (model.Radios, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.Err {
		return nil, errors.New("Error!")
	}
	return m.All, nil
}

func (m *MockedRadioRepo) Put(radio *model.Radio) error {
	if m.Err {
		return errors.New("error")
	}
	if radio.ID == "" {
		radio.ID = id.NewRandom()
	}
	m.Data[radio.ID] = radio
	return nil
}
