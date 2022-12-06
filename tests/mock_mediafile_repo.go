package tests

import (
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
)

func CreateMockMediaFileRepo() *MockMediaFileRepo {
	return &MockMediaFileRepo{
		data: make(map[string]*model.MediaFile),
	}
}

type MockMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]*model.MediaFile
	err  error
}

func (m *MockMediaFileRepo) SetError(err error) {
	m.err = err
}

func (m *MockMediaFileRepo) SetData(mfs model.MediaFiles) {
	m.data = make(map[string]*model.MediaFile)
	for i, mf := range mfs {
		m.data[mf.ID] = &mfs[i]
	}
}

func (m *MockMediaFileRepo) Exists(id string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	if m.err != nil {
		return nil, m.err
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	if m.err != nil {
		return nil, m.err
	}
	var res model.MediaFiles
	for _, a := range m.data {
		res = append(res, *a)
	}
	return res, nil
}

func (m *MockMediaFileRepo) Put(mf *model.MediaFile) error {
	if m.err != nil {
		return m.err
	}
	if mf.ID == "" {
		mf.ID = uuid.NewString()
	}
	m.data[mf.ID] = mf
	return nil
}

func (m *MockMediaFileRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.err != nil {
		return m.err
	}
	if d, ok := m.data[id]; ok {
		d.PlayCount++
		d.PlayDate = timestamp
		return nil
	}
	return model.ErrNotFound
}

var _ model.MediaFileRepository = (*MockMediaFileRepo)(nil)
