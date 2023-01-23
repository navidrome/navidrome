package tests

import (
	"errors"

	"github.com/navidrome/navidrome/model"
)

type MockedRadioInfoRepo struct {
	model.RadioInfoRepository
	err     bool
	data    map[string]*model.RadioInfo
	Options model.QueryOptions
}

func CreateMockedRadioInfoRepo() *MockedRadioInfoRepo {
	return &MockedRadioInfoRepo{
		data: make(map[string]*model.RadioInfo),
	}
}

func (m *MockedRadioInfoRepo) SetError(err bool) {
	m.err = err
}

func (m *MockedRadioInfoRepo) CountAll(options ...model.QueryOptions) (int64, error) {
	if m.err {
		return 0, errors.New("error")
	}
	return int64(len(m.data)), nil
}

func (m *MockedRadioInfoRepo) GetAll(...model.QueryOptions) (model.RadioInfos, error) {
	if m.err {
		return nil, errors.New("error")
	}

	var all model.RadioInfos
	for _, g := range m.data {
		all = append(all, *g)
	}
	return all, nil
}

func (m *MockedRadioInfoRepo) Get(id string) (*model.RadioInfo, error) {
	if m.err {
		return nil, errors.New("error")
	}

	for _, r := range m.data {
		if r.ID == id {
			return r, nil
		}
	}

	return nil, model.ErrNotFound
}

func (m *MockedRadioInfoRepo) Insert(model *model.RadioInfo) error {
	if m.err {
		return errors.New("error")
	}
	m.data[model.ID] = model
	return nil
}

func (m *MockedRadioInfoRepo) Purge() error {
	if m.err {
		return errors.New("error")
	}

	m.data = make(map[string]*model.RadioInfo)
	return nil
}

func (m *MockedRadioInfoRepo) Read(id string) (interface{}, error) {
	if m.err {
		return nil, errors.New("error")
	}
	return m.Get(id)
}
