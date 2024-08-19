package tests

import (
	"errors"
	"maps"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils/slice"
)

func CreateMockMediaFileRepo() *MockMediaFileRepo {
	return &MockMediaFileRepo{
		data: make(map[string]*model.MediaFile),
	}
}

type MockMediaFileRepo struct {
	model.MediaFileRepository
	data map[string]*model.MediaFile
	err  bool
}

func (m *MockMediaFileRepo) SetError(err bool) {
	m.err = err
}

func (m *MockMediaFileRepo) SetData(mfs model.MediaFiles) {
	m.data = make(map[string]*model.MediaFile)
	for i, mf := range mfs {
		m.data[mf.ID] = &mfs[i]
	}
}

func (m *MockMediaFileRepo) Exists(id string) (bool, error) {
	if m.err {
		return false, errors.New("error")
	}
	_, found := m.data[id]
	return found, nil
}

func (m *MockMediaFileRepo) Get(id string) (*model.MediaFile, error) {
	if m.err {
		return nil, errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		return d, nil
	}
	return nil, model.ErrNotFound
}

func (m *MockMediaFileRepo) GetAll(...model.QueryOptions) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}
	values := slices.Collect(maps.Values(m.data))
	return slice.Map(values, func(p *model.MediaFile) model.MediaFile {
		return *p
	}), nil
}

func (m *MockMediaFileRepo) Put(mf *model.MediaFile) error {
	if m.err {
		return errors.New("error")
	}
	if mf.ID == "" {
		mf.ID = uuid.NewString()
	}
	m.data[mf.ID] = mf
	return nil
}

func (m *MockMediaFileRepo) IncPlayCount(id string, timestamp time.Time) error {
	if m.err {
		return errors.New("error")
	}
	if d, ok := m.data[id]; ok {
		d.PlayCount++
		d.PlayDate = &timestamp
		return nil
	}
	return model.ErrNotFound
}

func (m *MockMediaFileRepo) FindByAlbum(artistId string) (model.MediaFiles, error) {
	if m.err {
		return nil, errors.New("error")
	}
	var res = make(model.MediaFiles, len(m.data))
	i := 0
	for _, a := range m.data {
		if a.AlbumID == artistId {
			res[i] = *a
			i++
		}
	}

	return res, nil
}

var _ model.MediaFileRepository = (*MockMediaFileRepo)(nil)
