package tests

import (
	"errors"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
)

type MockedRadioRepo struct {
	model.RadioRepository
	data    map[string]*model.Radio
	all     model.Radios
	err     bool
	Options model.QueryOptions
}

func CreateMockedRadioRepo() *MockedRadioRepo {
	return &MockedRadioRepo{}
}

func (m *MockedRadioRepo) SetError(err bool) {
	m.err = err
}

func (m *MockedRadioRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.err {
		return 0, errors.New("error")
	}
	return int64(len(m.data)), nil
}

func (m *MockedRadioRepo) Delete(id string) error {
	if m.err {
		return errors.New("Error!")
	}

	_, found := m.data[id]

	if !found {
		return errors.New("not found")
	}

	delete(m.data, id)
	return nil
}

func (m *MockedRadioRepo) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("Error!")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockedRadioRepo) Get(id string) (*model.Radio, error) {
	if m.err {
		return nil, errors.New("Error!")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockedRadioRepo) GetAll(qo ...model.QueryOptions) (model.Radios, error) {
	if len(qo) > 0 {
		m.Options = qo[0]
	}
	if m.err {
		return nil, errors.New("Error!")
	}
	return m.all, nil
}

func (m *MockedRadioRepo) Put(radio *model.Radio) error {
	if m.err {
		return errors.New("error")
	}
	if radio.ID == "" {
		radio.ID = uuid.NewString()
	}
	m.data[radio.ID] = radio
	return nil
}
